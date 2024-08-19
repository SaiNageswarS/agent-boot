import os
import unittest
from dotenv import load_dotenv
from src.personalized_response_generator import PersonalizedResponseGenerator
from src.clients.embedding import EmbeddingInterface, OpenAIEmbedding
from tests.mock_embedding import MiniLmEmbedding
from src.clients.vectordb import VectorDbInterface, PineconeDb
from src.knowledge_base import KnowledgeBase
from tests.mock_vector_db import MockVectorDb

load_dotenv()


class TestPersonalizedResponse(unittest.TestCase):
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

        self.kb.index_query(kb_data["qhash"], kb_data["question"], kb_data)
        self.personalized_response_gen = PersonalizedResponseGenerator(kb=self.kb)

    def test_personalized_response_withScore(self):
        context = {
            "User Profile": '''{ "name": "Sai Nageswar S", "gender": "male", "age": 36 }''',
            "User Test Scores": sample_score,
        }

        result = self.personalized_response_gen.generate(
            query="Explain subject constitution and which topics in constitution should I focus.",
            context=context,
            kb_query="Indian constitution and articles",
            threshold=0.5
        )
        self.assertIsNotNone(result)

    def test_personalized_response_knowledgeQuery(self):
        context = {}

        result = self.personalized_response_gen.generate(
            query="What is the procedure to amend constitution",
            context=context,
            kb_query="constitution amendment procedure",
            threshold=0.50
        )
        self.assertIsNotNone(result)


sample_score = '''{"accuracy":0.7045454545454546,"total":88,"correct":62,"subjectAccuracy":[{"total":7,"correct":7,"topicAccuracy":[{"topic":"Indian Heritage and Culture","accuracy":1,"total":1,"correct":1},{"topic":"History of India and Indian National Movement","accuracy":1,"total":2,"correct":2},{"topic":"Geography of the World and Society","accuracy":0,"total":0,"correct":0},{"topic":"Indian Society and Social Issues","accuracy":1,"total":4,"correct":4}],"subject":"General Studies Paper I","accuracy":1},{"subject":"General Studies Paper II","accuracy":0.5,"total":12,"correct":6,"topicAccuracy":[{"topic":"Governance","accuracy":1,"total":2,"correct":2},{"total":4,"correct":3,"topic":"Constitution of India","accuracy":0.75},{"accuracy":0,"total":0,"correct":0,"topic":"Polity"},{"topic":"Social Justice","accuracy":1,"total":1,"correct":1},{"accuracy":0,"total":5,"correct":0,"topic":"International Relations"}]},{"topicAccuracy":[{"topic":"Economic Development","accuracy":0,"total":0,"correct":0},{"topic":"Technology","accuracy":1,"total":3,"correct":3},{"topic":"Biodiversity","accuracy":0,"total":1,"correct":0},{"topic":"Security","accuracy":0,"total":0,"correct":0},{"topic":"Disaster Management","accuracy":0,"total":0,"correct":0}],"subject":"General Studies Paper III","accuracy":0.75,"total":4,"correct":3},{"total":10,"correct":5,"topicAccuracy":[{"correct":1,"topic":"Ethics and Human Interface","accuracy":0.25,"total":4},{"correct":0,"topic":"Attitude","accuracy":0,"total":2},{"topic":"Aptitude and Foundational Values","accuracy":0,"total":0,"correct":0},{"topic":"Emotional Intelligence","accuracy":0,"total":0,"correct":0},{"accuracy":1,"total":4,"correct":4,"topic":"Public/Civil Service Values and Ethics in Public Administration"}],"subject":"General Studies Paper IV","accuracy":0.5},{"subject":"Optional Subject: History","accuracy":0.6666666666666666,"total":6,"correct":4,"topicAccuracy":[{"topic":"Ancient Indian History","accuracy":1,"total":2,"correct":2},{"topic":"Medieval Indian History","accuracy":1,"total":2,"correct":2},{"correct":0,"topic":"Modern Indian History","accuracy":0,"total":1},{"total":1,"correct":0,"topic":"World History","accuracy":0}]},{"accuracy":0.8235294117647058,"total":17,"correct":14,"topicAccuracy":[{"topic":"Physical Geography","accuracy":1,"total":1,"correct":1},{"topic":"Human Geography","accuracy":0.75,"total":4,"correct":3},{"total":6,"correct":5,"topic":"Geography of India","accuracy":0.8333333333333334},{"topic":"Contemporary Issues","accuracy":0.8333333333333334,"total":6,"correct":5}],"subject":"Optional Subject: Geography"},{"subject":"Optional Subject: Public Administration","accuracy":0.42857142857142855,"total":7,"correct":3,"topicAccuracy":[{"accuracy":0,"total":1,"correct":0,"topic":"Administrative Theory"},{"topic":"Indian Administration","accuracy":0,"total":2,"correct":0},{"topic":"Public Policy","accuracy":0,"total":0,"correct":0},{"topic":"Governance and Good Governance","accuracy":0.75,"total":4,"correct":3}]},{"subject":"Optional Subject: Sociology","accuracy":0.5,"total":4,"correct":2,"topicAccuracy":[{"topic":"Fundamentals of Sociology","accuracy":0,"total":1,"correct":0},{"topic":"Sociology of India","accuracy":0,"total":0,"correct":0},{"topic":"Social Research and Methods","accuracy":0,"total":0,"correct":0},{"accuracy":0.6666666666666666,"total":3,"correct":2,"topic":"Contemporary Social Issues"}]},{"subject":"Optional Subject: Literature (Various Languages)","accuracy":0.7777777777777778,"total":9,"correct":7,"topicAccuracy":[{"topic":"History of Language and Literature","accuracy":1,"total":5,"correct":5},{"correct":0,"topic":"Classical and Modern Literature","accuracy":0,"total":0},{"topic":"Literary Criticism","accuracy":1,"total":2,"correct":2},{"topic":"Comparative Literature","accuracy":0,"total":2,"correct":0}]},{"subject":"Optional Subject: Political Science & International Relations","accuracy":0.9166666666666666,"total":12,"correct":11,"topicAccuracy":[{"topic":"Political Theory and Thought","accuracy":1,"total":5,"correct":5},{"correct":2,"topic":"Comparative Politics and Political Analysis","accuracy":1,"total":2},{"topic":"International Relations","accuracy":0,"total":0,"correct":0},{"topic":"Indian Government and Politics","accuracy":0.8,"total":5,"correct":4}]}]}'''

kb_data = {
    "qhash": "02754e411f00a9941e9ba08a4d59a589",
    "question": "Which of the following articles of the Indian Constitution deals with the 'Right to Equality'?",
    "explanation": "The Right to Equality is covered under Articles 14 to 18 of the Indian Constitution. These articles ensure equality before the law, prohibit discrimination on various grounds, and abolish practices such as untouchability and titles, thereby promoting social equality.",
    "subject": "General Studies Paper II",
    "topic": "Constitution of India",
    "answer": "A) Article 14 to 18"
}
