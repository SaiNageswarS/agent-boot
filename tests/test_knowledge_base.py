import unittest
from src.knowledge_base import index_query, get_nearest_neighbors, delete_query
from dotenv import load_dotenv

load_dotenv()


class TestKnowledgeBase(unittest.TestCase):
    def test_insert_query_delete(self):
        for record in sample_data:
            index_query(record["qhash"], record["question"], record)

        result = get_nearest_neighbors("foundational text for classical Indian dance")
        # only one similar query should be fetched.
        self.assertEquals(1, len(result))
        self.assertEquals("35f3984ee34a33753d328786b2e3e980", result[0]["qhash"])
        delete_query([x["qhash"] for x in sample_data])

    def test_insert_similar_query(self):
        for record in sample_data:
            is_indexed = index_query(record["qhash"], record["question"], record)
            self.assertEquals(True, is_indexed)

        similar_data = {
            "explanation": "Bharat Natyam is classical dance in state of Tamil Nadu.",
            "subject": "General Studies Paper I",
            "qhash": "sim_q1",
            "question": "which of the following is classical Indian dance in state of Tamil Nadu?",
            "answer": "C) Bharat Natyam",
            "topic": "Indian Heritage and Culture",
        }

        is_indexed = index_query(similar_data["qhash"], similar_data["question"], similar_data)
        self.assertEquals(True, is_indexed)

        similar_data2 = {
            "explanation": "The Natya Shastra, attributed to the sage Bharata Muni, is an ancient Indian treatise that deals with the performing arts, including theatre, dance, and music. It is considered the foundational text for classical Indian dance and drama, laying down the principles and techniques for performance and production.",
            "subject": "General Studies Paper I",
            "qhash": "sim_q2",
            "question": "Which of the following Indian texts is primarily associated with classical Indian dance?",
            "answer": "C) Natya Shastra",
            "topic": "Indian Heritage and Culture",
        }

        is_indexed = index_query(similar_data2["qhash"], similar_data2["question"], similar_data2)
        self.assertEquals(False, is_indexed)

        delete_query([x["qhash"] for x in sample_data])
        delete_query(["sim_q1", "sim_q2"])


sample_data = [
    {
      "explanation": "The Natya Shastra, attributed to the sage Bharata Muni, is an ancient Indian treatise that deals with the performing arts, including theatre, dance, and music. It is considered the foundational text for classical Indian dance and drama, laying down the principles and techniques for performance and production.",
      "subject": "General Studies Paper I",
      "qhash": "35f3984ee34a33753d328786b2e3e980",
      "question": "Which of the following ancient Indian texts is primarily associated with the performance and theory of classical Indian dance and drama?",
      "answer": "C) Natya Shastra",
      "topic": "Indian Heritage and Culture",
    },

    {
        "qhash": "eb1679d7ea18fcd258d849c52e407b18",
        "question": "Who was the founder of the Indian National Congress (INC) and in which year was it established?",
        "answer": "A) A.O. Hume, 1885",
        "subject": "General Studies Paper I",
        "topic": "History of India and Indian National Movement",
        "explanation": "The Indian National Congress (INC) was founded by Allan Octavian Hume, a retired British civil servant, in 1885. The INC played a significant role in the Indian independence movement against British rule.",
    }
]
