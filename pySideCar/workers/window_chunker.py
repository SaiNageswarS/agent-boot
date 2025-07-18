import logging
import gc
from collections.abc import Iterator

import spacy
import tiktoken
from workers.indexer_types import Chunk

# Set up logging
logging.basicConfig(
    level=logging.INFO, format="%(asctime)s - %(levelname)s - %(message)s"
)
logger = logging.getLogger(__name__)


class WindowChunker:
    def __init__(self):
        self.nlp = spacy.load("en_core_web_sm", disable=["ner", "tagger", "lemmatizer"])
        if "sentencizer" not in self.nlp.pipe_names:  # fail-safe
            self.nlp.add_pipe("sentencizer")

        self.nlp.max_length = 1_000_000  # Set a high limit for large texts
        self.encoding = tiktoken.get_encoding("cl100k_base")

    def chunk_windows(
        self, section_chunk: Chunk, window_size: int = 700, stride: int = 600
    ) -> Iterator[Chunk]:
        """
        Splits a list of section chunks into smaller windows of 10 chunks each.

        Args:
            section_chunks (list[Chunk]): List of section chunks to be split.

        Returns:
            Iterator[Chunk]: Enumerable of windowed chunks.
            • No sentence is ever split across windows.
            • `window_size` ≈ max tokens per window.
            • Windows overlap by ~`window_size - stride` tokens
            (aligned to sentence boundaries).
        """

        # Split the body into sentences
        logger.info(
            "Processing section chunk: %s with %d characters.",
            section_chunk.chunkId,
            len(section_chunk.sentences[0])
        )
        
        sentences = self._split_sentences(section_chunk.sentences[0])
        logger.info(
            f"Found {len(sentences)} sentences in section chunk: {section_chunk.chunkId}."
        )
        sent_tok_lens = [self._count_tokens(sent) for sent in sentences]

        start_sent = 0
        w_idx = 0

        while start_sent < len(sentences):
            tok_cnt, end_sent = 0, start_sent

            # Grow window until adding the next sentence would exceed the budget
            while (
                end_sent < len(sentences)
                and tok_cnt + sent_tok_lens[end_sent] <= window_size
            ):
                tok_cnt += sent_tok_lens[end_sent]
                end_sent += 1

            # Edge case: a *single* very long sentence
            if end_sent == start_sent:
                tok_cnt = sent_tok_lens[start_sent]
                end_sent = start_sent + 1

            window_sentences = sentences[start_sent:end_sent]

            # Create a new Chunk object for the window
            yield Chunk(
                chunkId=f"{section_chunk.chunkId}_{w_idx}",
                sectionPath=section_chunk.sectionPath,
                sectionIndex=section_chunk.sectionIndex,
                title=section_chunk.title,
                sourceUri=section_chunk.sourceUri,
                sentences=window_sentences,
                sectionId=section_chunk.sectionId,
                windowIndex=w_idx,
                prevChunkId="",
                nextChunkId="",
            )

            w_idx += 1

            # Advance start_sent by ≈ stride tokens, but always land on a sentence boundary
            stride_tok = 0
            while (
                start_sent < end_sent
                and stride_tok + sent_tok_lens[start_sent] < stride
            ):
                stride_tok += sent_tok_lens[start_sent]
                start_sent += 1

            # If stride is so large we emptied the window, move at least one sentence
            if start_sent == end_sent:
                start_sent += 1


        gc.collect()

    def _count_tokens(self, text: str) -> int:
        """
        Counts the number of tokens in a given text using the tiktoken encoding.

        Args:
            text (str): The text to count tokens for.

        Returns:
            int: The number of tokens in the text.
        """
        return len(self.encoding.encode(text)) if text else 0

    def _split_sentences(self, text: str) -> list[str]:
        """
        Splits a text into sentences using the spaCy sentencizer.

        Args:
            text (str): The text to split into sentences.

        Returns:
            list[str]: List of sentences.
        """
        if len(text) <= self.nlp.max_length:
            # Small enough: single pass
            return [sent.text.strip() for sent in self.nlp(text).sents]

        result = []

        pos = 0
        # Split text into manageable chunks
        while pos < len(text):
            chunk_end = min(pos + self.nlp.max_length, len(text))
            chunk = text[pos:chunk_end]

            doc = self.nlp(chunk)
            chunk_sentences = list(doc.sents)

            if chunk_end >= len(text):
                # Last chunk, no need to check for next sentence
                result.extend([sent.text.strip() for sent in chunk_sentences])
                break
            
            if len(chunk_sentences) <= 1:
                # If only one sentence, add it and move to next chunk. 
                # Otherwise, we might loop forever.
                result.append(chunk_sentences[0].text.strip())
                pos = chunk_end
            else:
                # Take all but last sentence
                result.extend([sent.text.strip() for sent in chunk_sentences[:-1]])

                # Find where last sentence starts in the original text
                last_sent = chunk_sentences[-1]
                last_sent_start = last_sent.start_char  # Position within chunk
                pos = pos + last_sent_start  # Absolute position in text

        return result
