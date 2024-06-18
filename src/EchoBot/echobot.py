from botbuilder.core import TurnContext
from botbuilder.schema import ChannelAccount

class EchoBot:
    async def on_turn(self, turn_context: TurnContext):
        if turn_context.activity.type == "message":
            # Echo the received message
            await turn_context.send_activity(turn_context.activity.text)
        elif turn_context.activity.type == "conversationUpdate":
            if turn_context.activity.members_added:
                for member in turn_context.activity.members_added:
                    if member.id != turn_context.activity.recipient.id:
                        await turn_context.send_activity("Welcome to the bot!")
