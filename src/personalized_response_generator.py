"""
Uses llama 8b model to give a personalized response to user query.
Personalization will be based on context. Use **few_shot_intent_classification** to derive required context.
"""

import json
from llama_index.llms.groq import Groq
from llama_index.core.llms import ChatMessage

from src.knowledge_base import KnowledgeBase

system_prompt_template = """
You are a helpful assistant and your task is to answer user query based on below context. Consider the user's 
background, preferences, and goals to tailor the response. The response should be informative, empathetic, 
and actionable.

**Response Guidelines**
- Address the user by name and acknowledge their query.
- Provide accurate and relevant information to answer the query.
- Consider the user's background, goals, and preferences to personalize the response.
- Offer actionable advice or next steps when possible.
- Keep the response concise and easy to understand
"""

user_query_template = """
Query: {query}

**Context**
{context}
"""


class PersonalizedResponseGenerator:
    def __init__(self, kb: KnowledgeBase = None):
        self.kb = kb or KnowledgeBase()

    def generate(self, query: str, context: dict[str, str], kb_query: str, threshold=0.75) -> str:
        print(f"Generating personalized response for {query}")
        llm = Groq(model="llama3-8b-8192")
        system_prompt = system_prompt_template

        if __is_not_empty_or_null__(kb_query):
            print(f"Querying KB for {kb_query}")
            kb_results = self.kb.get_nearest_neighbors(query=kb_query, threshold=threshold)
            if len(kb_results) > 0:
                context["knowledge"] = json.dumps(kb_results)

        context_str = __get_context_from_dict__(context)

        user_query_prompt = user_query_template.format(query=query, context=context_str)

        messages = [
            ChatMessage(role="system", content=system_prompt),
            ChatMessage(role="user", content=user_query_prompt),
        ]

        resp = llm.chat(messages)
        result = resp.message.content
        return result


def __get_context_from_dict__(context: dict[str, str]) -> str:
    result = ""

    for key, value in context.items():
        result += f"- {key}: {value}\n"

    return result


def __is_not_empty_or_null__(s: str) -> bool:
    return bool(s and s.strip())
