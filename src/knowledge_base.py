import os
from pinecone import Pinecone
from openai import OpenAI


def index_query(qid: str, query: str, metadata: dict[str, str], skip_similar=True, threshold=0.9) -> bool:
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
    embedding = __get_embedding__(query)
    index = __get_index__()

    if skip_similar:
        similar = __get_nearest_neighbors_by_embedding__(embedding, threshold, 3)
        if len(similar) > 0:
            return False

    index.upsert(
        vectors=[
            {
                "id": qid,
                "values": embedding,
                "metadata": metadata
            }
        ]
    )

    return True


def get_nearest_neighbors(query: str, threshold=0.75, top_k=3):
    """
    Get nearest neighbors.
    :param query: Query to get similar entries.
    :param threshold: Filter similar queries by input threshold.
    :param top_k: Max similar entries to retrieve.
    :return:
    """
    embedding = __get_embedding__(query)
    return __get_nearest_neighbors_by_embedding__(embedding, threshold, top_k)


def __get_nearest_neighbors_by_embedding__(embedding: list[float], threshold: float, top_k: int):
    index = __get_index__()

    result = index.query(
        vector=embedding,
        top_k=top_k,
        include_values=True,
        include_metadata=True
    )

    result = [x.metadata for x in result.matches if x.score >= threshold]
    return result


def delete_query(qids: list[str]):
    """
    Delete query from index.
    :param qids: List of query Ids to delete.
    :return:
    """
    index = __get_index__()
    index.delete(ids=qids)


def __get_index__():
    pinecone_key = os.getenv("PINECONE_KEY")
    pc = Pinecone(api_key=pinecone_key)

    # agent-boot users have to create "agent_boot_kb" vector index in pinecone.
    index = pc.Index("agent-boot-kb")

    return index


def __get_embedding__(query: str) -> list[float]:
    client = OpenAI(api_key=os.getenv("OPENAI_API_KEY"))

    query = query.replace("\n", " ")
    embedding = client.embeddings.create(
        input=[query],
        model="text-embedding-3-small").data[0].embedding

    return embedding
