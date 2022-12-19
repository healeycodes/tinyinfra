import sys
import subprocess
import requests

proc = subprocess.Popen(
    ["go", "run", "."], stdout=subprocess.PIPE, stderr=subprocess.STDOUT
)

addr = "http://localhost:8000"
if len(sys.argv) > 1:
    addr = sys.argv[1]

try:
    # new user
    user_new = requests.post(f"{addr}/user/new")
    assert user_new.status_code == 200
    token = user_new.json()["token"]
    assert token

    headers = {"Authorization": f"Bearer {token}"}

    # set key
    key_name = "a"
    key_value = "b"
    kv_set = requests.post(
        f"{addr}/kv/set", headers=headers, json={"key": key_name, "value": key_value}
    )
    assert kv_set.status_code == 200

    # get key
    kv_get = requests.post(
        f"{addr}/kv/get", headers=headers, json={"key": key_name, "value": key_value}
    )
    assert kv_get.status_code == 200
    assert kv_get.json()["key"] == key_name
    assert kv_get.json()["value"] == key_value
    assert kv_get.json()["ttl"] == -1

    # send queue item
    namespace = "a"
    message = "b"
    queue_send = requests.post(
        f"{addr}/queue/send",
        headers=headers,
        json={"namespace": namespace, "message": message},
    )
    assert queue_send.status_code == 200

    # receive queue item
    queue_receive = requests.post(
        f"{addr}/queue/receive",
        headers=headers,
        json={"namespace": namespace, "visibilityTimeout": 20000},
    )
    assert queue_receive.status_code == 200
    assert queue_receive.json()["id"]
    assert queue_receive.json()["namespace"] == namespace
    assert queue_receive.json()["message"] == message

    # delete queue item
    namespace = "a"
    message = "b"
    queue_delete = requests.post(
        f"{addr}/queue/delete",
        headers=headers,
        json={"namespace": namespace, "id": queue_receive.json()["id"]},
    )
    assert queue_delete.status_code == 200

    print("tests pass 🚀")
except Exception as e:
    print("tests fail 🛑")
    raise
finally:
    proc.terminate()
