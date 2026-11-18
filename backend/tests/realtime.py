"""Real local RabbitMQ -> Centrifugo integration; demo accounts only."""
from registration_mailbox import register as verified_registration
import asyncio, json, os, subprocess, urllib.request, urllib.error, uuid
import websockets

def request(path,data,token=None):
 headers={'Content-Type':'application/json','X-Saya-Client':'native'}
 if token:headers['Authorization']='Bearer '+token
 request=urllib.request.Request('http://localhost:8080/api/v1'+path,data=json.dumps(data).encode(),headers=headers)
 with urllib.request.urlopen(request,timeout=15) as response:return response.status,json.load(response)
def api(path,data,token=None):
 return request(path,data,token)[1]
async def main():
 status,auth=verified_registration(request,{'email':str(uuid.uuid4())+'@example.test','password':'demo-password-long-enough'})
 bearer=auth['access_token'];uid=auth['user']['id'];channel='user:'+uid+':assessments'
 connection=api('/realtime/connection-token',{},bearer)['token']
 subscription=api('/realtime/subscription-token',{'channel':channel},bearer)['token']
 try:api('/realtime/subscription-token',{'channel':'user:'+str(uuid.uuid4())+':assessments'},bearer);raise AssertionError('foreign channel allowed')
 except urllib.error.HTTPError as error:assert error.code==401
 async with websockets.connect('ws://localhost:8000/connection/websocket',origin='http://localhost:3000') as denied:
  await denied.send(json.dumps({'id':1,'connect':{'token':connection}}));assert 'connect' in json.loads(await denied.recv())
  await denied.send(json.dumps({'id':2,'subscribe':{'channel':channel}}));assert 'error' in json.loads(await denied.recv())
  await denied.send(json.dumps({'id':3,'subscribe':{'channel':'user:'+str(uuid.uuid4())+':assessments','token':subscription}}))
  try:
   result=json.loads(await denied.recv());assert 'error' in result,'mismatched channel allowed'
  except websockets.exceptions.ConnectionClosedError as error:
   assert error.rcvd.code==3500
 async with websockets.connect('ws://localhost:8000/connection/websocket',origin='http://localhost:3000') as ws:
  await ws.send(json.dumps({'id':1,'connect':{'token':connection}}));result=json.loads(await ws.recv());assert 'connect' in result
  await ws.send(json.dumps({'id':2,'subscribe':{'channel':channel,'token':subscription}}));result=json.loads(await ws.recv());assert 'subscribe' in result
  env=dict(os.environ,DEMO_USER_ID=uid)
  process=await asyncio.create_subprocess_exec('docker','compose','--env-file','backend/deployments/local/.env','-f','backend/deployments/local/compose.yaml','run','--rm','--build','demo',env=env,stdout=asyncio.subprocess.DEVNULL,stderr=asyncio.subprocess.DEVNULL)
  try:
   while True:
    raw=json.loads(await asyncio.wait_for(ws.recv(),timeout=120))
    if 'push' in raw:
     event=raw['push']['pub']['data'];assert event['event']=='demo.completed',event;assert set(event)=={'event','assessment_id','status','version'};break
    if raw=={}:await ws.send('{}')
   print('PASS authenticated connection, own subscription, foreign subscription denied, RabbitMQ -> Centrifugo event')
  finally:await process.wait()
asyncio.run(main())
