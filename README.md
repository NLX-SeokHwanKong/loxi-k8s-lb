# loxi-k8s-lb
loxi-k8s-lb
---

## 1. Compile 

```
$ make release
```

## 2. Start LOXI-LB CCM

```
$ cd manifests

$ kubectl apply -f netlox-ccm.yaml
```

## 3. Start Nginx Deployment & Service

```
$ cd manifests

$ kubectl apply -f nginx-deploy.yaml

$ kubectl apply -f netlox-ccm.yaml

```