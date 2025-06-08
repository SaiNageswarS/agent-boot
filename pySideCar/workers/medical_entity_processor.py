import spacy, scispacy
from scispacy.linking import EntityLinker
import logging

from workers.indexer_types import Chunk


# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


class MedicalEntityProcessor:
    """
    Processes medical chunks using SciSpacy + UMLS for entity recognition and linking.
    Implements research-backed strategies for medical text processing.
    """

    def __init__(self, 
                 model_name: str = "en_core_sci_sm",
                 linker_name: str = "umls",
                 confidence_threshold: float = 0.85):
        """
        Initialize the medical entity processor.
        
        Args:
            model_name: SciSpacy model to use
            linker_name: UMLS linker name
            confidence_threshold: Minimum confidence for entity linking
        """
        self.confidence_threshold = confidence_threshold
        
        # Load SciSpacy model
        logger.info(f"Loading SciSpacy model: {model_name}")
        self.nlp = spacy.load(model_name)
        
            
        # Add UMLS entity linker
        logger.info(f"Adding UMLS linker: {linker_name}")
        self.nlp.add_pipe("scispacy_linker", config={
            "linker_name": linker_name,
            "threshold": confidence_threshold
        })
        
        self.linker = self.nlp.get_pipe("scispacy_linker")
        logger.info("Medical entity processor initialized successfully")

    
    def process_chunk(self, chunk: Chunk) -> Chunk:
        """
        Process a single chunk with SciSpacy + UMLS linking.

        Args:
            chunk: Input chunk to process
            
        Returns:
            EnrichedChunk with medical entity information
        """
        logger.debug(f"Processing chunk: {chunk.chunkId}")

        # Process main chunk body
        doc = self.nlp(chunk.body)
        entities = self._extract_entities(doc, chunk.body)
        
        # Process section path for additional context
        section_text = " > ".join(chunk.sectionPath)
        section_doc = self.nlp(section_text)
        section_entities = self._extract_entities(section_doc, section_text)
            
        # Create enriched chunk with combined entities and abbreviations
        chunk.tags = entities + section_entities
        
        return chunk
    

    def _extract_entities(self, doc, text: str) -> list[str]:
        """Extract and link medical entities from processed document."""
        entities = []
        
        for ent in doc.ents:
            # Get UMLS linking information
            cui, score, canonical_name = None, None, None
            semantic_types, aliases, definition, sources = [], [], None, []
            
            if ent._.kb_ents:
                # Get best linking candidate
                best_candidate = ent._.kb_ents[0]
                cui = best_candidate[0]
                score = best_candidate[1]
            
            # Apply confidence-based filtering
            if score is None or score >= self.confidence_threshold:
                medical_tag = cui if cui else ent.text
                entities.append(medical_tag)
        
        return entities

