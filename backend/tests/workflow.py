"""P1 local-only demo scenario, ownership and idempotency."""
import os, json, urllib.request, urllib.error, uuid, time

def call(path,token=None,body=None):
 headers={'Content-Type':'application/json','X-Saya-Client':'native'}
 if token:headers['Authorization']='Bearer '+token
 request=urllib.request.Request(os.environ.get('SAYA_API_BASE','http://localhost:8080')+'/api/v1'+path,data=None if body is None else json.dumps(body).encode(),headers=headers)
 try:
  with urllib.request.urlopen(request,timeout=15)as response:return response.status,json.loads(response.read()or b'{}')
 except urllib.error.HTTPError as error:return error.code,{}
def register():
 status,data=call('/auth/register',body={'email':str(uuid.uuid4())+'@example.test','password':'demo-password-with-entropy'})
 assert status==201,status
 return data['access_token']
token=register();other=register()
code,survey=call('/surveys/current',token);assert code==200,code
body={'questionnaire_id':survey['id'],'version':survey['version'],'answers':{'immediate_threat':True,'information_requested':True},'consent':True,'external_llm_consent':False,'idempotency_key':str(uuid.uuid4())}
assert call('/surveys/submissions',token,dict(body,consent=False))[0]==400
code,operation=call('/surveys/submissions',token,body);assert code==202,code
code,repeat=call('/surveys/submissions',token,body);assert code==202 and repeat['id']==operation['id']
assert call('/surveys/submissions',token,dict(body,answers={'immediate_threat':False,'information_requested':True}))[0]==400
assert call('/assessments/'+operation['id'],other)[0]==404
for i in range(60):
 code,result=call('/assessments/'+operation['id'],token);assert code==200,code
 if result['status']!='pending':break
 time.sleep(1)
assert result['status']=='completed',result
assert result['provider']=='demonstration-mock',result
assert result['immediate_threat_reported'] is True
assert len(result['recommendations'])>=2
print('PASS survey consent, persisted submission, idempotency/conflict, ownership, async RabbitMQ/AI, deterministic rule, completed result')
