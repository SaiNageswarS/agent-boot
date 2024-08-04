from abc import ABC, abstractmethod
import os
from pinecone import Pinecone


class VectorDbInterface(ABC):
    @abstractmethod
    def get_nearest_neighbors(self, embedding: list[float], threshold: float, top_k: int):
        pass

    @abstractmethod
    def upsert(self, qid: str, embedding: list[float], metadata: dict[str, str]):
        pass

    @abstractmethod
    def delete(self, qids: list[str]):
        pass


class PineconeDb(VectorDbInterface):
    def __init__(self, pinecone_key: str = None, index_name: str = None):
        pinecone_key = pinecone_key or os.getenv("PINECONE_KEY")
        index_name = index_name or os.getenv("INDEX_NAME")

        pc = Pinecone(api_key=pinecone_key)
        self.index = pc.Index(index_name)

    def get_nearest_neighbors(self, embedding: list[float], threshold: float, top_k: int):
        result = self.index.query(
            vector=embedding,
            top_k=top_k,
            include_values=True,
            include_metadata=True
        )

        result = [x.metadata for x in result.matches if x.score >= threshold]
        return result

    def upsert(self, qid: str, embedding: list[float], metadata: dict[str, str]):
        return self.index.upsert(
            vectors=[
                {
                    "id": qid,
                    "values": embedding,
                    "metadata": metadata
                }
            ]
        )

    def delete(self, qids: list[str]):
        return self.index.delete(ids=qids)
