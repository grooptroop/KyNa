
Оглавление:

1. [Генерация ключа к ВМ](#Генирация-ключа-к-ВМ)
2. [Разворачивание инфраструктуры](#Разворачивание-инфраструктуры)


----------------------------------------------------------------------------------------

## Генирация ключа к ВМ 

Для начала на машине с которой мы будем выполнять плейбуки создадим ключи

```
ssh-keygen -t ed25519 -f ~/.ssh/my_ansible_key
```

Затем публичный ключ загружаем на сервер (на которым мы будем разворачивать инфру)

```
ssh-copy-id -i ~/.ssh/my_ansible_key.pub user@server_ip
```


проверим достучимся ли мы до хоста(выполняем в корне проекта)
```
ansible -i k3s_builder_stand/inventories/hosts.ini k3s_server -m ping
```

должны увидеть что то вроде:
```
k3s-worker | SUCCESS => {
    "ansible_facts": {
        "discovered_interpreter_python": "/usr/bin/python3.12"
    },
    "changed": false,
    "ping": "pong"
}
```

## Разворачивание инфраструктуры
Заходим в k3s_builder_stand/inventories/hosts.ini и меняем на наши хосты и данные
```
[k3s_server:vars]
ansible_ssh_private_key_file=~/.ssh/my_ansible_key
ansible_user=user

[k3s_server]
k3s-worker ansible_host=192.168.1.70 
```

Просто переходим в папку с ansible playbook и запускаем его
```
cd k3s_builder_stand 

ansible-playbook playbooks/main.yml
```

проверяем есть ли namespace и поды 

```
kubectl get namespaces

kubectl get pods -n (имя namespace)
```