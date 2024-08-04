from src.clients.embedding import EmbeddingInterface, OpenAIEmbedding
from src.clients.vectordb import VectorDbInterface, PineconeDb


class KnowledgeBase:
    def __init__(self, embedding: EmbeddingInterface = None, vector_db: VectorDbInterface = None):
        self.embedding = embedding or OpenAIEmbedding()
        self.vector_db = vector_db or PineconeDb()

    def index_query(self, qid: str, query: str, metadata: dict[str, str], skip_similar=True, threshold=0.9) -> bool:
        """
        Add query to knowledge base.
        :param qid: Primary Identifier of query. Used to delete or fetch by ID.
        :param query: Query to index in Vector Index.
        :param metadata:
        :param skip_similar: Skip indexing if similar entry exists in knowledge base. Helps in reducing
            number of entries in knowledge base
        :param threshold: Threshold of similarity to skip indexing.
        :return: True if indexing was successful else False
        """
        embedding = self.embedding.get_embedding(query)

        if skip_similar:
            try:
                similar = self.vector_db.get_nearest_neighbors(embedding, threshold, 3)
                if len(similar) > 0:
                    print(f"Similar items found existing {similar}")
                    return False
            except Exception as e:
                print(f"Failed getting similar items. Error: {e}")

        self.vector_db.upsert(qid=qid, embedding=embedding, metadata=metadata)
        print("Indexed successfully")

        return True

    def get_nearest_neighbors(self, query: str, threshold=0.75, top_k=3):
        """
        Get nearest neighbors.
        :param query: Query to get similar entries.
        :param threshold: Filter similar queries by input threshold.
        :param top_k: Max similar entries to retrieve.
        :return:
        """
        embedding = self.embedding.get_embedding(query)
        return self.vector_db.get_nearest_neighbors(embedding, threshold, top_k)

    def delete_query(self, qids: list[str]):
        """
        Delete query from index.
        :param qids: List of query Ids to delete.
        :return:
        """
        self.vector_db.delete(qids=qids)
