"""Only operates on the disposable saya-verify stack."""
from registration_mailbox import register as verified_registration
import json,os,subprocess,time,urllib.request,urllib.error,uuid
assert os.environ.get('SAYA_DISPOSABLE_TEST')=='1'
compose=['docker','compose','-p','saya-verify','--env-file','/tmp/saya-verify/deployments/local/.env','-f','backend/deployments/local/compose.yaml','-f','/tmp/saya-verify/override.yaml']
def command(args,env=None):
 result=subprocess.run(compose+args,env=env,capture_output=True)
 assert result.returncode==0,'disposable compose operation failed'
def call(path,body=None,token=None):
 headers={'Content-Type':'application/json','X-Saya-Client':'native'}
 if token:headers['Authorization']='Bearer '+token
 req=urllib.request.Request('http://localhost:18080/api/v1'+path,data=None if body is None else json.dumps(body).encode(),headers=headers)
 try:
  with urllib.request.urlopen(req,timeout=10)as response:return response.status,json.load(response)
 except urllib.error.HTTPError as error:return error.code,{}
def submit(token):
 body={'questionnaire_id':'saya-demo','version':1,'answers':{'immediate_threat':True,'information_requested':True},'consent':True,'external_llm_consent':False,'idempotency_key':str(uuid.uuid4())}
 code,result=call('/surveys/submissions',body,token);assert code==202,code
 return result['id']
def wait(token,id,target):
 for _ in range(100):
  try:
   code,result=call('/assessments/'+id,token=token)
   if code==200 and result['status']==target:return result
  except Exception:pass
  time.sleep(1)
 raise AssertionError('assessment did not reach '+target)
command(['up','-d','--build','survey','assessment'])
code,auth=verified_registration(call,{'email':str(uuid.uuid4())+'@example.test','password':'demo-failure-recovery-password'});assert code==201
token=auth['access_token']
external={**os.environ,'LLM_PROVIDER':'http','LLM_ENDPOINT':'https://example.invalid/demo','LLM_API_KEY':'demo-no-network'}
try:
 command(['up','-d','--no-deps','ai'],external)
 result=wait(token,submit(token),'failed')
 assert result['immediate_threat_reported'] and len(result['recommendations'])==1
 print('PASS failed assessment persisted with deterministic rule; missing consent prevents external request')
finally:command(['up','-d','--no-deps','ai'])
command(['stop','ai'])
id=submit(token)
command(['stop','rabbitmq'])
time.sleep(3)
command(['start','rabbitmq'])
command(['up','-d','--no-deps','ai'])
result=wait(token,id,'completed')
assert result['provider']=='demonstration-mock'
print('PASS queued assessment survives RabbitMQ restart and completes after recovery')
