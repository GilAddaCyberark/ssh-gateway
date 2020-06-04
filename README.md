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
    ssh gadda@ec2-user@i-06316bc63aea813ec@localhost -p 2233
    ssh gadda@gil@ec2-54-75-3-176.eu-west-1.compute.amazonaws.com:2022@localhost -p 2233

Future Ideas - 
Multi cloud instance id tokens:
e.g. 
aws#i-06316bc63aea813ec
gcp#some-instance_id
azr#another_ms_cloud_machine

log to EWS , monitor suspicious commnands
