#!/bin/bash
# set -euo pipefail
# include the common-used shortcuts
source libs.sh

# load .env.config cache and read previously used cluster_name
[ -f /tmp/.env.config ] && cat /tmp/.env.config && source /tmp/.env.config

echo $1

taintNode(){
    kubectl taint nodes "$1" SpotTerminating=true:NoExecute 
}

getNodeNameByInstanceId(){
    aws ec2 describe-instances --instance-id "$1" --query 'Reservations[0].Instances[0].NetworkInterfaces[0].PrivateDnsName' --output text
}

getClusterNameFromTags(){
    x=$(aws ec2 describe-tags --filters Name=resource-id,Values="$1" Name=value,Values=owned Name=resource-type,Values=instance --query "Tags[0].Key" --output text)
    echo ${x##*/}
}


detailType=$(echo $1 | jq -r '.["detail-type"] | select(type == "string")')
instanceId=$(echo $1 | jq -r '.detail.EC2InstanceId | select(type == "string")')
autoScalingGroupName=$(echo $1 | jq -r '.detail.AutoScalingGroupName | select(type == "string")')
lifecycleActionToken=$(echo $1 | jq -r '.detail.LifecycleActionToken | select(type == "string")')
lifecycleHookName=$(echo $1 | jq -r '.detail.LifecycleHookName | select(type == "string")')
lifecycleTransition=$(echo $1 | jq -r '.detail.LifecycleTransition | select(type == "string")')

input_cluster_name=$(getClusterNameFromTags $instanceId)

if [ -n "${input_cluster_name}" ]  && [ "${input_cluster_name}" != "${cluster_name}" ]; then
    echo "got new cluster_name=$input_cluster_name - update kubeconfig now..."
    update_kubeconfig "$input_cluster_name" || exit 1
    cluster_name="$input_cluster_name"
    echo "writing new cluster_name=${cluster_name} to /tmp/.env.config"
    echo "cluster_name=${cluster_name}" > /tmp/.env.config
fi

# taint the node immediately
echo "[NFO] taintNode now"
nodeName=$(getNodeNameByInstanceId $instanceId)
taintNode "$nodeName"
echo "[INFO] sleep a while before we callback the hook so the pods have enough time for resheduling"
sleep 10
echo "[INFO] OK. let's kubectl descirbe node/${nodeName}"
kubectl describe node/${nodeName}


if [ "$detailType"=="EC2 Instance-terminate Lifecycle Action" ]; then
    echo "start autoscaling group complete-lifecycle-actiopn callback"
    aws autoscaling complete-lifecycle-action \
    --lifecycle-hook-name $lifecycleHookName \
    --auto-scaling-group-name $autoScalingGroupName \
    --instance-id $instanceId \
    --lifecycle-action-token $lifecycleActionToken \
    --lifecycle-action-result "CONTINUE"
fi

exit 0