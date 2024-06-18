from flask import Flask,request,Response
from botbuilder.schema import Activity
from botbuilder.core import BotFrameworkAdapter,BotFrameworkAdapterSettings
import asyncio
from echobot import EchoBot

app = Flask(__name__)
loop = asyncio.get_event_loop()

botadaptersettings = BotFrameworkAdapterSettings("b08411fb-31bd-4a47-9468-4929cda96046","izM8Q~6932XiWQUzQ44yf8nCkeVU-a5Up0y3zbOd")
botadapter = BotFrameworkAdapter(botadaptersettings)

ebot = EchoBot()

@app.route('/api/messages',methods=['POST'])
def messages():
    if "application/json" in request.headers['Content-Type']:
        jsonmessage = request.json
    else:
        return Response(status=415)
    
    activity = Activity().deserialize(jsonmessage)

    async def turn_call(turn_context):
        await ebot.on_turn(turn_context)

    task = loop.create_task(botadapter.process_activity(activity,"",turn_call))
    loop.run_until_complete(task)

if __name__ == '__main__':
    app.run('localhost',3978)