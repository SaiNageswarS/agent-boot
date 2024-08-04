import unittest
from src.intent_classification import few_shot_intent_classification, __generate_prompt__, IntentExample
from dotenv import load_dotenv

load_dotenv()


class TestIntentClassification(unittest.TestCase):
    def test_generate_prompt(self):
        generated_prompt = __generate_prompt__(test_intent_examples)
        self.assertEqual(expected_prompt_template, generated_prompt)

    def test_few_shot_intent_classification_withScore(self):
        result = few_shot_intent_classification("How should I set by preparation schedule?", test_intent_examples)
        self.assertIsNotNone(result)
        self.assertIn("scores", result.contextRequired)

    def test_few_shot_intent_classification_withWeb(self):
        result = few_shot_intent_classification("When did India win football world cup last time?", test_intent_examples)
        self.assertIsNotNone(result)
        self.assertTrue(len(result.contextRequired) > 0)
        self.assertTrue(result.contextRequired[0].startswith("web result"))

    def test_few_shot_intent_classification_withWebAndScore(self):
        result = few_shot_intent_classification("Explain subject constitution and which topics in constitution should I focus.", test_intent_examples)
        self.assertIsNotNone(result)
        self.assertTrue(len(result.contextRequired) > 1)
        self.assertIn("scores", result.contextRequired)


test_intent_examples = [
    IntentExample(query="What are my scores?", contextRequired=["user_profile", "scores"],
                  interpretation="To convey personalized user scores, scores of user and profile are required"),

    IntentExample(query="How should I prepare for UPSC?",
                  contextRequired=["user_profile", "scores", "web result: UPSC syllabus"],
                  interpretation="To help user prepare for test, current user scores are required and syllabus for test is required from web."),

    IntentExample(query="Which river flows from east to west?",
                  contextRequired=["web result: river flowing from east to west"],
                  interpretation="Factual question which is required from the web."),

    IntentExample(query="Civil disobidience movement in India",
                  contextRequired=["web result: Civil disobidience movement in India"],
                  interpretation="Factual question which is required from the web."),

    IntentExample(query="Which subjects should I focus on?",
                  contextRequired=["user_profile", "scores", "web result: UPSC syllabus"],
                  interpretation="To suggest subjects to focus, current user scores are required to know user's strengths and weakness. Further, UPSC syllabus is required to suggest study schedule.")
]

expected_prompt_template = """
You are a context expert, and your task is to identify the necessary context required to answer a user's query. 
You will be provided with a query, please respond in the following format, replacing placeholders with actual content.

query placeholder | list of context required separated by [SEP] placeholder | interpretation placeholder.

**Important Guidelines**:

1. Be specific and concise in your responses.
2. Identify the minimum necessary context required to answer the query.
3. Consider both user-specific information (e.g., user_profile, scores) and external knowledge (e.g., web results) that may be needed to provide a relevant answer.
4. Provide a clear and concise interpretation of why the specified context is required to answer the query.
5. Use [SEP] separator for list of context required.

Here are some examples of how this prompt can be used:

**Query:** What are my scores?
**Response:**
What are my scores? | user_profile[SEP]scores | To convey personalized user scores, scores of user and profile are required

**Query:** How should I prepare for UPSC?
**Response:**
How should I prepare for UPSC? | user_profile[SEP]scores[SEP]web result: UPSC syllabus | To help user prepare for test, current user scores are required and syllabus for test is required from web.

**Query:** Which river flows from east to west?
**Response:**
Which river flows from east to west? | web result: river flowing from east to west | Factual question which is required from the web.

**Query:** Civil disobidience movement in India
**Response:**
Civil disobidience movement in India | web result: Civil disobidience movement in India | Factual question which is required from the web.

**Query:** Which subjects should I focus on?
**Response:**
Which subjects should I focus on? | user_profile[SEP]scores[SEP]web result: UPSC syllabus | To suggest subjects to focus, current user scores are required to know user's strengths and weakness. Further, UPSC syllabus is required to suggest study schedule.

"""
