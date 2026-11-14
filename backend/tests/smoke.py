"""Local disposable demo account; never run against production."""
import os, json, urllib.request, urllib.error, http.cookiejar, uuid
jar=http.cookiejar.CookieJar()
client=urllib.request.build_opener(urllib.request.HTTPCookieProcessor(jar))
base=os.environ.get('SAYA_API_BASE','http://localhost:8080')
def call(path, data=None, token=None, csrf=False):
 headers={}
 if token: headers['Authorization']='Bearer '+token
 if data is not None: headers['Content-Type']='application/json'
 if csrf: headers['X-CSRF-Token']=next(c.value for c in jar if c.name=='saya_csrf')
 req=urllib.request.Request(base+path,data=None if data is None else json.dumps(data).encode(),headers=headers)
 try:
  with client.open(req,timeout=15) as r: return r.status,json.loads(r.read() or b'{}')
 except urllib.error.HTTPError as e:return e.code,{}
credentials={'email':str(uuid.uuid4())+'@example.test','password':'demonstration-only-password'}
assert call('/api/v1/auth/register',dict(credentials,password='short'))[0]==400
credentials['role']='ADMIN'
code,body=call('/api/v1/auth/register',credentials);assert code==201,code
assert body['user']['role']=='USER'
assert call('/api/v1/auth/register',credentials)[0]==409
assert call('/api/v1/auth/login',dict(credentials,password='incorrect'))[0]==401
code,body=call('/api/v1/auth/login',credentials);assert code==200
assert call('/api/v1/users/me',token=body['access_token'])[0]==200
assert call('/api/v1/users/me',token='invalid')[0]==401
assert call('/api/v1/auth/refresh',{})[0]==403
old_refresh=next(c.value for c in jar if c.name=='saya_refresh')
code,rotated=call('/api/v1/auth/refresh',{},csrf=True);assert code==200,code
new_refresh=next(c.value for c in jar if c.name=='saya_refresh')
assert old_refresh!=new_refresh
for cookie in jar:
 if cookie.name=='saya_refresh':cookie.value=old_refresh
assert call('/api/v1/auth/refresh',{},csrf=True)[0]==401
for cookie in jar:
 if cookie.name=='saya_refresh':cookie.value=new_refresh
assert call('/api/v1/auth/refresh',{},csrf=True)[0]==401
assert call('/api/v1/auth/logout',{},csrf=True)[0]==204
assert call('/api/v1/auth/refresh',{})[0]==403
print('PASS register, duplicate, invalid password, login, protected profile, CSRF, rotation, logout')
