from src.clients.vectordb import VectorDbInterface
import numpy as np


class MockVectorDb(VectorDbInterface):
    def __init__(self):
        self.qid_to_embedding = {}  # Maps qid to embedding
        self.qid_to_metadata = {}  # Maps qid to metadata

    def get_nearest_neighbors(self, embedding: list[float], threshold: float, top_k: int):
        results = []
        for qid, stored_embedding in self.qid_to_embedding.items():
            similarity = self._cosine_similarity(embedding, stored_embedding)
            if similarity > threshold:
                results.append({
                    'qid': qid,
                    'similarity': similarity,
                    'metadata': self.qid_to_metadata[qid]
                })

        # Sort results by distance and return the top_k
        results.sort(key=lambda x: x['similarity'], reverse=True)
        results = results[:top_k]
        return [x['metadata'] for x in results]

    def upsert(self, qid: str, embedding: list[float], metadata: dict[str, str]):
        self.qid_to_embedding[qid] = embedding
        self.qid_to_metadata[qid] = metadata

    def delete(self, qids: list[str]):
        for qid in qids:
            if qid in self.qid_to_embedding:
                del self.qid_to_embedding[qid]
                del self.qid_to_metadata[qid]

    def _cosine_similarity(self, vec1: list[float], vec2: list[float]) -> float:
        # Convert lists to numpy arrays for vector operations
        vec1 = np.array(vec1)
        vec2 = np.array(vec2)

        # Compute the dot product and norms of the vectors
        dot_product = np.dot(vec1, vec2)
        norm_vec1 = np.linalg.norm(vec1)
        norm_vec2 = np.linalg.norm(vec2)

        # Compute cosine similarity
        cosine_similarity = dot_product / (norm_vec1 * norm_vec2)

        return cosine_similarity
