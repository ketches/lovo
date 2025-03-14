# lovo

lovo (Local Volume) 实现基于 kubernetes local 类型的存储卷的动态配置。

## 安装

```bash
kubectl apply -f https://raw.githubusercontent.com/ketches/lovo/master/deploy/manifests.yaml
```

## 示例

```bash
kubectl apply -f https://raw.githubusercontent.com/ketches/lovo/master/examples/mysql.yaml
```

验证，创建成功后会自动创建一个 `local` 类型的 PV。

```bash
kubectl get pv -l lovo.ketches.cn/pvc-namespace=default,lovo.ketches.cn/pvc-name=mysql-data -w
```

运行以上命令，等待 PV 创建成功，并且状态最终更新为 `Bound`。

查看 PV 的详细信息：

```bash
kubectl describe pv pvc-<pvc-uid>
```

可以看到 PV 包含两个 Annotation：

- `lovo.ketches.cn/node`：表明 PV 的存储节点。
- `lovo.ketches.cn/path=/var/lib/lovo/<pvc-namespace>/<pvc-uid>`，表明是 PV 在节点上的存储位置。

清理示例：

```bash
kubectl delete -f https://raw.githubusercontent.com/ketches/lovo/master/examples/mysql.yaml
```

## 卸载

```bash
kubectl delete -f https://raw.githubusercontent.com/ketches/lovo/master/deploy/manifests.yaml
```
