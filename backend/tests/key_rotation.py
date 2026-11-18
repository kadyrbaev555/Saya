"""Destructive CONFIG test for disposable saya-verify project only; no production target."""
from registration_mailbox import register as verified_registration
import base64,json,os,pathlib,subprocess,time,urllib.request,urllib.error,uuid
root=pathlib.Path('/tmp/saya-verify')
envfile=root/'deployments/local/.env'
assert os.environ.get('SAYA_DISPOSABLE_TEST')=='1','Explicit disposable test flag required'
assert envfile.is_file()
seedroot=pathlib.Path('/tmp/saya-keyrotation');(seedroot/'deployments/local').mkdir(parents=True,exist_ok=True)
if not(seedroot/'deployments/local/.env').exists():
 result=subprocess.run(['/tmp/saya-dev-init'],cwd=seedroot,capture_output=True)
 assert result.returncode==0,'key generation failed'
def read(path):return dict(line.split('=',1)for line in path.read_text().splitlines()if '='in line and not line.startswith('#'))
values=read(envfile);new=read(seedroot/'deployments/local/.env')
base='http://localhost:18080'
def request(path,body=None,token=None):
 headers={'Content-Type':'application/json','X-Saya-Client':'native'}
 if token:headers['Authorization']='Bearer '+token
 req=urllib.request.Request(base+path,data=None if body is None else json.dumps(body).encode(),headers=headers)
 try:
  with urllib.request.urlopen(req,timeout=10)as response:return response.status,json.load(response)
 except urllib.error.HTTPError as e:return e.code,{}
def account():
 code,session=verified_registration(lambda path, data: request('/api/v1'+path,data),{'email':str(uuid.uuid4())+'@example.test','password':'demonstration-key-rotation'})
 assert code==201,code
 return session['access_token']
def apply(services):
 envfile.write_text(''.join(k+'='+v+'\n'for k,v in values.items()))
 result=subprocess.run(['docker','compose','-p','saya-verify','--env-file',str(envfile),'-f','backend/deployments/local/compose.yaml','-f','/tmp/saya-verify/override.yaml','up','-d','--no-deps']+services,capture_output=True)
 assert result.returncode==0,'compose configuration update failed'
 for attempt in range(30):
  try:
   with urllib.request.urlopen(base+'/health/ready',timeout=2)as response:
    if response.status==200:return
  except Exception:time.sleep(.3)
 raise AssertionError('gateway readiness timeout')
old=account();oldkey=values['JWT_PUBLIC']
values['JWT_SEED']=new['JWT_SEED'];values['JWT_PUBLIC']=new['JWT_PUBLIC'];values['JWT_KID']='rotated-2';values['JWT_PUBLIC_KEYS']=json.dumps({'local-1':oldkey,'rotated-2':new['JWT_PUBLIC']},separators=(',',':'))
apply(['auth','gateway'])
assert request('/api/v1/users/me',token=old)[0]==200,'retiring token rejected'
active=account();assert json.loads(base64.urlsafe_b64decode(active.split('.')[0]+'=='))['kid']=='rotated-2'
assert request('/api/v1/users/me',token=active)[0]==200
assert len(request('/.well-known/jwks.json')[1]['keys'])==2
values['JWT_PUBLIC_KEYS']=json.dumps({'rotated-2':new['JWT_PUBLIC']},separators=(',',':'));apply(['gateway'])
assert request('/api/v1/users/me',token=old)[0]==401
assert request('/api/v1/users/me',token=active)[0]==200
print('PASS new signer, overlap validation, two-key JWKS, retired-key rejection')
