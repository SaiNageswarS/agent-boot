import unittest
from dotenv import load_dotenv
from src.personalized_response import personalized_response_generator

load_dotenv()


class TestPersonalizedResponse(unittest.TestCase):
    def test_personalized_response_withScore(self):
        context = {
            "User Profile": '''{ "name": "Sai Nageswar S", "gender": "male", "age": 36 }''',
            "User Test Scores": sample_score,
        }

        result = personalized_response_generator(
            query="Explain subject constitution and which topics in constitution should I focus.",
            context=context,
            kb_query="Indian constitution and articles"
        )
        self.assertIsNotNone(result)

    def test_personalized_response_knowledgeQuery(self):
        context = {}

        result = personalized_response_generator(
            query="What is the procedure to amend constitution",
            context=context,
            kb_query="constitution amendment procedure",
            threshold=0.50
        )
        self.assertIsNotNone(result)


sample_score = '''{"accuracy":0.7045454545454546,"total":88,"correct":62,"subjectAccuracy":[{"total":7,"correct":7,"topicAccuracy":[{"topic":"Indian Heritage and Culture","accuracy":1,"total":1,"correct":1},{"topic":"History of India and Indian National Movement","accuracy":1,"total":2,"correct":2},{"topic":"Geography of the World and Society","accuracy":0,"total":0,"correct":0},{"topic":"Indian Society and Social Issues","accuracy":1,"total":4,"correct":4}],"subject":"General Studies Paper I","accuracy":1},{"subject":"General Studies Paper II","accuracy":0.5,"total":12,"correct":6,"topicAccuracy":[{"topic":"Governance","accuracy":1,"total":2,"correct":2},{"total":4,"correct":3,"topic":"Constitution of India","accuracy":0.75},{"accuracy":0,"total":0,"correct":0,"topic":"Polity"},{"topic":"Social Justice","accuracy":1,"total":1,"correct":1},{"accuracy":0,"total":5,"correct":0,"topic":"International Relations"}]},{"topicAccuracy":[{"topic":"Economic Development","accuracy":0,"total":0,"correct":0},{"topic":"Technology","accuracy":1,"total":3,"correct":3},{"topic":"Biodiversity","accuracy":0,"total":1,"correct":0},{"topic":"Security","accuracy":0,"total":0,"correct":0},{"topic":"Disaster Management","accuracy":0,"total":0,"correct":0}],"subject":"General Studies Paper III","accuracy":0.75,"total":4,"correct":3},{"total":10,"correct":5,"topicAccuracy":[{"correct":1,"topic":"Ethics and Human Interface","accuracy":0.25,"total":4},{"correct":0,"topic":"Attitude","accuracy":0,"total":2},{"topic":"Aptitude and Foundational Values","accuracy":0,"total":0,"correct":0},{"topic":"Emotional Intelligence","accuracy":0,"total":0,"correct":0},{"accuracy":1,"total":4,"correct":4,"topic":"Public/Civil Service Values and Ethics in Public Administration"}],"subject":"General Studies Paper IV","accuracy":0.5},{"subject":"Optional Subject: History","accuracy":0.6666666666666666,"total":6,"correct":4,"topicAccuracy":[{"topic":"Ancient Indian History","accuracy":1,"total":2,"correct":2},{"topic":"Medieval Indian History","accuracy":1,"total":2,"correct":2},{"correct":0,"topic":"Modern Indian History","accuracy":0,"total":1},{"total":1,"correct":0,"topic":"World History","accuracy":0}]},{"accuracy":0.8235294117647058,"total":17,"correct":14,"topicAccuracy":[{"topic":"Physical Geography","accuracy":1,"total":1,"correct":1},{"topic":"Human Geography","accuracy":0.75,"total":4,"correct":3},{"total":6,"correct":5,"topic":"Geography of India","accuracy":0.8333333333333334},{"topic":"Contemporary Issues","accuracy":0.8333333333333334,"total":6,"correct":5}],"subject":"Optional Subject: Geography"},{"subject":"Optional Subject: Public Administration","accuracy":0.42857142857142855,"total":7,"correct":3,"topicAccuracy":[{"accuracy":0,"total":1,"correct":0,"topic":"Administrative Theory"},{"topic":"Indian Administration","accuracy":0,"total":2,"correct":0},{"topic":"Public Policy","accuracy":0,"total":0,"correct":0},{"topic":"Governance and Good Governance","accuracy":0.75,"total":4,"correct":3}]},{"subject":"Optional Subject: Sociology","accuracy":0.5,"total":4,"correct":2,"topicAccuracy":[{"topic":"Fundamentals of Sociology","accuracy":0,"total":1,"correct":0},{"topic":"Sociology of India","accuracy":0,"total":0,"correct":0},{"topic":"Social Research and Methods","accuracy":0,"total":0,"correct":0},{"accuracy":0.6666666666666666,"total":3,"correct":2,"topic":"Contemporary Social Issues"}]},{"subject":"Optional Subject: Literature (Various Languages)","accuracy":0.7777777777777778,"total":9,"correct":7,"topicAccuracy":[{"topic":"History of Language and Literature","accuracy":1,"total":5,"correct":5},{"correct":0,"topic":"Classical and Modern Literature","accuracy":0,"total":0},{"topic":"Literary Criticism","accuracy":1,"total":2,"correct":2},{"topic":"Comparative Literature","accuracy":0,"total":2,"correct":0}]},{"subject":"Optional Subject: Political Science & International Relations","accuracy":0.9166666666666666,"total":12,"correct":11,"topicAccuracy":[{"topic":"Political Theory and Thought","accuracy":1,"total":5,"correct":5},{"correct":2,"topic":"Comparative Politics and Political Analysis","accuracy":1,"total":2},{"topic":"International Relations","accuracy":0,"total":0,"correct":0},{"topic":"Indian Government and Politics","accuracy":0.8,"total":5,"correct":4}]}]}'''