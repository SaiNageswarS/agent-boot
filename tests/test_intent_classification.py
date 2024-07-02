import unittest
from src.intent_classification import few_shot_intent_classification
from dotenv import load_dotenv

load_dotenv()


class TestIntentClassification(unittest.TestCase):
    def test_few_shot_intent_classification_withScore(self):
        result = few_shot_intent_classification("What subjects should I focus today")
        self.assertIsNotNone(result)
        self.assertIn("scores", result.context)

    def test_few_shot_intent_classification_withWeb(self):
        result = few_shot_intent_classification("When did India win football world cup last time?")
        self.assertIsNotNone(result)
        self.assertTrue(len(result.context) > 0)
        self.assertTrue(result.context[0].startswith("web result"))

