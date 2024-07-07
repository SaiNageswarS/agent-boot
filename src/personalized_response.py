"""
Uses llama 8b model to give a personalized response to user query.
Personalization will be based on context. Use **few_shot_intent_classification** to derive required context.
"""

from llama_index.llms.groq import Groq
from llama_index.core.llms import ChatMessage

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
- User Profile: {userProfile}
{otherContext}
"""


def personalized_response_generator(query: str, user_profile_json: str, other_context: str) -> str:
    llm = Groq(model="llama3-8b-8192")
    system_prompt = system_prompt_template
    user_query_prompt = user_query_template.format(query=query, userProfile=user_profile_json, otherContext=other_context)

    messages = [
        ChatMessage(role="system", content=system_prompt),
        ChatMessage(role="user", content=user_query_prompt),
    ]

    resp = llm.chat(messages)
    result = resp.message.content
    return result
