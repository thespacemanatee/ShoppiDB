import json
import requests 

PUT_ENDPOINT = "http://localhost:8000/put"

for i in range(1000): 
    data = json.dumps({'key' : "0", 'value': ''}).encode("utf-8")
    r = requests.post(url = PUT_ENDPOINT, data = data)
    print(r.text)
    