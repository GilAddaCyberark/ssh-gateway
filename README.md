# ssh-gateway
An SSH gateway to support an ssh bastion. Different type of authentication, ssh session recording
The SSH Gatewaty connects to a target in that method:
    Client App --> SSH Gateway --> Target Machine

To launch the ssh gatway  

1. go build 

2. execute : ./ssh-gateway -config config,json 

3. running a client vs the ssh gateway is done by: 
    ssh personal_user@target_user@target_address@bastion_address

    Example: 
    ssh gadda@gil@176.34.154.225:2025@localhost -p 2222
