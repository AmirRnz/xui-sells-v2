import json
import sys
import io

sys.stdout = io.TextIOWrapper(sys.stdout.buffer, encoding='utf-8', errors='replace')

with open('3x-ui_openapi.json', 'r', encoding='utf-8') as f:
    data = json.load(f)

print("OpenAPI version:", data.get('openapi'))

print("\n--- Security Schemes ---")
print(json.dumps(data.get('components', {}).get('securitySchemes', {}), indent=2))

print("\n--- Client Schema Properties ---")
client_props = data.get('components', {}).get('schemas', {}).get('Client', {}).get('properties', {})
for k, v in client_props.items():
    print(f"  {k}: type={v.get('type')}, format={v.get('format')}, example={v.get('example')}")

print("\n--- Subscription Links endpoints ---")
for path in ['/panel/api/clients/subLinks/{subId}', '/panel/api/clients/links/{email}', '/{subPath}{subid}']:
    if path in data.get('paths', {}):
        print(f"Path {path}: {json.dumps(data['paths'][path], indent=2)[:500]}")

print("\n--- All endpoints related to client / inbound / setting ---")
for path, methods in data.get('paths', {}).items():
    for method, op in methods.items():
        if any(w in path for w in ['client', 'inbound', 'setting', 'subPath', 'subLinks', 'subId']):
            print(f"{method.upper():6} {path:40} : {op.get('summary', '')[:70]}")
