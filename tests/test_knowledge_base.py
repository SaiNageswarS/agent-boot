import os
import unittest
from src.knowledge_base import KnowledgeBase
from src.clients.embedding import EmbeddingInterface, OpenAIEmbedding, MiniLmEmbedding
from src.clients.vectordb import VectorDbInterface, PineconeDb
from tests.mock_vector_db import MockVectorDb
from dotenv import load_dotenv

load_dotenv()


class TestKnowledgeBase(unittest.TestCase):
    def setUp(self):
        open_ai_api_key = os.getenv("OPENAI_API_KEY")
        pinecone_key = os.getenv("PINECONE_KEY")
        index_name = "agent-boot-kb-test"

        if open_ai_api_key and pinecone_key and index_name:
            print("Using OpenAI and pinecone")
            embedding: EmbeddingInterface = OpenAIEmbedding(api_key=open_ai_api_key)
            vector_db: VectorDbInterface = PineconeDb(pinecone_key=pinecone_key, index_name=index_name)
            self.kb = KnowledgeBase(embedding, vector_db)
        else:
            print("Using miniLM and mock_vector_db")
            embedding: EmbeddingInterface = MiniLmEmbedding()
            vector_db: VectorDbInterface = MockVectorDb()
            self.kb = KnowledgeBase(embedding, vector_db)

    def test_insert_query_delete(self):
        for record in sample_data:
            self.kb.index_query(record["qhash"], record["question"], record)

        result = self.kb.get_nearest_neighbors("foundational text for classical Indian dance")
        # only one similar query should be fetched.
        self.assertEqual(1, len(result))
        self.assertEqual("35f3984ee34a33753d328786b2e3e980", result[0]["qhash"])
        self.kb.delete_query([x["qhash"] for x in sample_data])

    def test_insert_similar_query(self):
        for record in sample_data:
            is_indexed = self.kb.index_query(record["qhash"], record["question"], record)
            self.assertEqual(True, is_indexed)

        similar_data = {
            "explanation": "Bharat Natyam is classical dance in state of Tamil Nadu.",
            "subject": "General Studies Paper I",
            "qhash": "sim_q1",
            "question": "which of the following is classical Indian dance in state of Tamil Nadu?",
            "answer": "C) Bharat Natyam",
            "topic": "Indian Heritage and Culture",
        }

        is_indexed = self.kb.index_query(similar_data["qhash"], similar_data["question"], similar_data)
        self.assertEqual(True, is_indexed)

        similar_data2 = {
            "explanation": "The Natya Shastra, attributed to the sage Bharata Muni, is an ancient Indian treatise that deals with the performing arts, including theatre, dance, and music. It is considered the foundational text for classical Indian dance and drama, laying down the principles and techniques for performance and production.",
            "subject": "General Studies Paper I",
            "qhash": "sim_q2",
            "question": "Which of the following Indian texts is primarily associated with classical Indian dance?",
            "answer": "C) Natya Shastra",
            "topic": "Indian Heritage and Culture",
        }

        is_indexed = self.kb.index_query(similar_data2["qhash"], similar_data2["question"], similar_data2)
        self.assertEqual(False, is_indexed)

        self.kb.delete_query([x["qhash"] for x in sample_data])
        self.kb.delete_query(["sim_q1", "sim_q2"])


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
    },

    {
        "answer": "C) Enhancing the performance of traditional fossil fuels",
        "subject": "General Studies Paper III",
        "topic": "Technology",
        "qhash": "fccfca1c17113deec90b35534ab19fc8",
        "question": "Which of the following is NOT a potential application of nanotechnology?",
        "explanation": "Nanotechnology is widely applied in various fields including medicine, energy, and electronics. It can be used for targeted drug delivery to improve medical treatments, for developing more efficient solar panels to harness solar energy, and for enhancing the efficiency of electrical grids through better materials and designs. However, enhancing the performance of traditional fossil fuels is not typically associated with nanotechnology advancements."
    },

    {
        "topic": "Technology",
        "question": "Which of the following technologies is primarily used to enhance the accuracy and efficiency of genome editing?",
        "subject": "General Studies Paper III",
        "answer": "a) CRISPR-Cas9",
        "explanation": "CRISPR-Cas9 is a revolutionary technology that allows for precise and efficient editing of DNA within genomes. This technology has significant applications in biotechnology, including research, medicine, and agriculture. It stands for Clustered Regularly Interspaced Short Palindromic Repeats (CRISPR) and CRISPR-associated protein 9 (Cas9).",
        "qhash": "574ade302555cee9ff9855f985cf0e56"
    },

    {
        "qhash": "07b760487924dba07f0e1968dfab6f38",
        "answer": "B. 89th Amendment Act, 2003",
        "topic": "Social Justice",
        "subject": "General Studies Paper II",
        "question": "Which constitutional amendment led to the establishment of the National Commission for Scheduled Castes, thereby promoting social justice?",
        "explanation": "The 89th Amendment Act of 2003 bifurcated the National Commission for Scheduled Castes and Scheduled Tribes into two separate commissions: the National Commission for Scheduled Castes and the National Commission for Scheduled Tribes. This amendment was significant in promoting social justice as it allowed for more focused attention on the issues and welfare of Scheduled Castes."
    },

    {
        "subject": "General Studies Paper I",
        "topic": "Geography of the World and Society",
        "createdBy": "RZ6yuW7y5mA=",
        "answer": "C. Tibetan Plateau",
        "explanation": "The Tibetan Plateau, also known as the 'Roof of the World,' is the world's highest and largest plateau, with an average elevation of over 4,500 meters above sea level. It is located in Central Asia, covering much of the Tibet Autonomous Region and Qinghai Province in China, as well as parts of India, Nepal, and Bhutan.",
        "qhash": "24e2d366d1437ec7663dbc2216c6b7d5",
        "question": "Which of the following is the highest plateau in the world?"
    }
]
