#!/bin/bash
# set -euo pipefail
# include the common-used shortcuts
source libs.sh

echo $1

taintNode(){
    kubectl taint nodes "$1" SpotTerminating=true:NoExecute 
}

drainNode(){
    kubectl drain  "$1" --ignore-daemonsets --delete-local-data
}

getNodeNameByInstanceId(){
    aws ec2 describe-instances --instance-id "$1" --query 'Reservations[0].Instances[0].NetworkInterfaces[0].PrivateDnsName' --output text
}

getClusterNameFromTags(){
    x=$(aws ec2 describe-tags --filters Name=resource-id,Values="$1" Name=value,Values=owned Name=resource-type,Values=instance --query "Tags[0].Key" --output text)
    echo ${x##*/}
}

update_kubeconfig(){
    aws eks update-kubeconfig --name "$1"  --kubeconfig /tmp/kubeconfig
}


detailType=$(echo $1 | jq -r '.["detail-type"] | select(type == "string")')
instanceId=$(echo $1 | jq -r '.detail.EC2InstanceId | select(type == "string")')
autoScalingGroupName=$(echo $1 | jq -r '.detail.AutoScalingGroupName | select(type == "string")')
lifecycleActionToken=$(echo $1 | jq -r '.detail.LifecycleActionToken | select(type == "string")')
lifecycleHookName=$(echo $1 | jq -r '.detail.LifecycleHookName | select(type == "string")')
lifecycleTransition=$(echo $1 | jq -r '.detail.LifecycleTransition | select(type == "string")')

# always get the cluster_name from EC2 Tag
input_cluster_name=$(getClusterNameFromTags $instanceId)

# always update kubeconfig
update_kubeconfig "$cluster_name" 

# drain the node immediately
echo "[INFO] start the node draining now"
nodeName=$(getNodeNameByInstanceId $instanceId)
if [ "${drain_type}" == "taint" ]; then
    echo "[INFO] start taint ${nodeName}"
    taintNode "$nodeName"
else
    echo "[INFO] start drain ${nodeName}"
    drainNode "$nodeName"
fi
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