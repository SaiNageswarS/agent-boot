from abc import ABC, abstractmethod
import os
from openai import OpenAI


class EmbeddingInterface(ABC):
    @abstractmethod
    def get_embedding(self, text: str) -> list[float]:
        """
        Generate an embedding for the given text.

        :param text: The text to generate an embedding for.
        :return: A list of floats representing the embedding.
        """
        pass


class OpenAIEmbedding(EmbeddingInterface):
    def __init__(self, api_key: str = None):
        api_key = api_key or os.getenv("OPENAI_API_KEY")
        if not api_key:
            raise ValueError("API key must be provided either as an argument or in the env variable 'OPENAI_API_KEY'")

        self.client = OpenAI(api_key=api_key)

    def get_embedding(self, query: str) -> list[float]:
        query = query.replace("\n", " ")
        embedding = self.client.embeddings.create(
            input=[query],
            model="text-embedding-3-small").data[0].embedding

        return embedding
