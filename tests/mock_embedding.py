from src.clients.embedding import EmbeddingInterface
from sentence_transformers import SentenceTransformer


class MiniLmEmbedding(EmbeddingInterface):
    def __init__(self):
        self.model = SentenceTransformer('all-MiniLM-L6-v2')

    def get_embedding(self, text: str) -> list[float]:
        embedding = self.model.encode(text).tolist()
        return embedding
