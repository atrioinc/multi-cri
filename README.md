# Multi-cri
Multi-cri is a modulable container runtime interface (CRI) for kubernetes which manages the pod lifecycle and allows to configure adapters for different container runtimes or resource managers such as SLURM.
In addition, it provides multi-CRI for kubernetes, so different CRIs can be configured by setting the RuntimeClass pod attribute.
## Kubernetes version
Multi-cri is implemented under Kubernetes v1.13.0
 
## Command
Multi-cri execution creates a unix socket. It can be configured by using the following options:

      --adapter-name                     Adapter name. It setup "slurm" by default. 
      --enable-pod-network               Enable pod network namespace
      --enable-pod-persistence           Enable pod and container persistence in cache file
      --network-bin-dir string           The directory for putting network binaries. (default "/opt/cni/bin")
      --network-conf-dir string          The directory for putting network plugin configuration files. (default "/etc/cni/net.d")
      --remote-runtime-endpoints         Remote runtime endpoints to support RuntimeClass. Add several by separating with comma. (default "default:/var/run/dockershim.sock")
      --resources-cache-path string      Path where image, container and sandbox information will be stored. It will also be the image pool path (default "/root/.multi-cri/")
      --root-dir string                  Root directory path for multi-cri managed files (metadata checkpoint etc). (default "/var/lib/multi-cri")
      --sandbox-image string             The image used by sandbox container. (default "gcr.io/google_containers/pause:3.0")
      --socket-path string               Path to the socket which multi-cri serves on. (default "/var/run/multicri.sock")
      --stderrthreshold severity         logs at or above this threshold go to stderr (default 2)
      --stream-addr string               The ip address streaming server is listening on. Default host interface is used if this is empty.
      --stream-port string               The port streaming server is listening on. (default "10010")

## Network Namespace
Multi-cri allows to configure pod network namespace by using CNI. It can be enabled by using `--enable-pod-network`.

## Network Namespace
Pod and container metadata can be persistent in disk by enabling it by using `--enable-pod-persistence`.

# Kubernetes multi CRI support
In order to provide multi-CRI for kubernetes, we support [RuntimeClass](https://kubernetes.io/docs/concepts/containers/runtime-class/) which can configure several remote CRIs and identify them with a name. It configures
the runtime in the `runtimeClassName` pod spec attribute. Our implementation we contemplate several scenarios:
- Configure a default remote CRI which will be used in case that `runtimeClassName` attribute does not have value. In this case, we will setup `runtimeClassName: multicri` in the pods we want to run in multi-cri.
- The default CRI is not configured, so multi-cri will be used by default.
- Several remote CRI endpoints will be configured. It can be done in this format: `--remote-runtime-endpoints default:/var/run/dockershim.sock`

Due to Kubernetes does not provide pod information to de image manager, we need to specify the runtime class to the image name: `image: multicri/perl`

We need to configure a full container runtime interface, able to execute docker containers. For, example dockershim.

In the following, we can see how to configure the Multicri runtimeClass:

```
# kubectl apply -f runtime_multicri.yaml

apiVersion: node.k8s.io/v1alpha1  # RuntimeClass is defined in the node.k8s.io API group
kind: RuntimeClass
metadata:
  name: multicri
  # The name the RuntimeClass will be referenced by
  # RuntimeClass is a non-namespaced resource
spec:
  runtimeHandler: multicri 

```


The last section of this document shows an example of the full multi-cri setup.

# Image specification
Our CRI provides support to docker and singularity repositories. In order to identify the images in the multi-cri runtimeClass, they need to start always with the selected CRI identification name ```multicri```.

## Singularity repository
This CRI supports the Singularity repository, `https://singularity-hub.org`, which contains a bunch of public singularity images.
Due to the kubernetes image format does not allow to include `://`, we need to specify the image without those characters. Therefore, we use `singularity-repository` to identify images from this kind of repository.

In the following, we show the image pull workflow for singularity hub images.
- **User specify the image name** including the repository name (`singularity-repository`) in kubernetes with format. For example: `sigularity-repository.jorgesece/singularity-nginx:latest`.
- **CRI parse the image name** to get the right singularity image url. For example it will generate the url `shub://jorgesece/singularity-nginx` from the aforementioned image name.
- **CRI builds the image** by using the right singularity client (singularity build `shub://jorgesece/singularity-nginx`),
 registers the  image in the CRI image pool and stores it in the image storage folder.
 
## Docker repository
Docker image repository is supported, at least for those adapters which are based on singularity or docker.

In the following, we show the image pull workflow for docker repository images:

- **User specify the image name** including the repository name (`docker-repository`). For example: `docker-repository.perl`.
- **CRI parse the image name** to get the right docker image url. For example it will generate the url `docker://perl`.
- **CRI builds the image** by using the singularity client (singularity build `docker://perl`),
registers the  image in the CRI image pool and stores it in the image storage folder.

It supports docker private respositories by using ImageSecret credentials.

## Local images
Users can upload their images in a container volume and specify the path to that image.
- **User specify the image name** including the repository name (`local-image`). For example: `local-image.perl.img`.
- **CRI parse the image name** to get the right image path in the volume. For example it will generate the path `volumen/perl.img`.
- **CRI copy the image** to the working user directory.

# Adapters
Multi-cri aims to be a generic CRI in which different runtimes are supported by implementing different adapters.
We can configure it by setting the `--adapter-name` variable.
At the moment, there is an adapter for the Slurm workload manager.


## Slurm adapter
Slurm adapter supports batch job submissions to Slurm clusters.

### Configuration
* **CRI_SLURM_MOUNT_PATH**: String  environment variable. It is the working directory in the Slurm cluster ("multi-cri" by default). This path is relative to the $HOME directory.
* **CRI_SLURM_IMAGE_REMOTE_MOUNT**: String environment variable. It is the path in which the images will be built (empty by default).
They are built in the container persistent volume path by default.
* **CRI_SLURM_BUILD_IN_CLUSTER**: Boolean environment variable which indicates to build images directly in the Slurm cluster (default false).
Images will build in the CRI node by default. 

### Features
- MPI jobs are supported. Configured by environment variables.
- Slurm cluster credentials are provided by environment variables.
- Data transfer supported by using NFS. Containers mount NFS volumes, which are linked to the proper Slurm NFS mount.
- Local image repository use images stored in the NFS container volume.

### Container environment variables
Container job execution are configured by the following environment variables:
* Slurm credentials:
  * **CLUSTER_USERNAME**: user name to access the cluster.
  * **CLUSTER_PASSWORD**: user password to access the cluster.
  * **CLUSTER_HOST**: host/ip related to the cluster.
* Slurm prerun configuration:
  * **CLUSTER_CONFIG**: Prerun script which will be executed before the run script defined by the container command. It must be passed as text.
* Slurm job configuration:
  * **JOB_QUEUE**: queue in which submit the job.
  * **JOB_GPU**: GPU configuration. Format "gpu[[:type]:count]". For instance: `gpu:kepler:2`. More information [Slurm GRES](https://slurm.schedmd.com/gres.html)
  * **JOB_NUM_NODES**: number of nodes.
  * **JOB_NUM_CORES_NODE**: number of cores in each node.
  * **JOB_NUM_CORES**: number of cores to distribute through the nodes.
  * **JOB_NUM_TASKS_NODE**: num of tasks to allocate in one node.
  * **JOB_CUSTOM_CONFIG**: custom Slurm environment variables. More information in [Slurm input environment variables](https://slurm.schedmd.com/sbatch.html).
 
* MPI configuration: 
  * **MPI_VERSION**: MPI version. It is considered as MPI job when it has value. In case it is not set, the job won't be MPI.
  * **MPI_FLAGS**: MPI flags.

Note: Container environment variables with **CLUSTER_***, **JOB_***, **KUBERNETES_*** and **MPI_FLAGS** pattern are reserved to the system.
 
### NFS configuration
In order to properly work with SLURM, we must to configure the NFS in this way:
* K8s side
  * Create NFS PersistentVolume(PV) and PersistentVolumeClaim(PVC) to the NFS path (`/<NFS PATH>`).
  * Mount the volume in container with `mountPath: "multicri"`. So the CRI will know which is the Slurm volume.
* Slurm side
  * Mount the NFS path, `/<NFS PATH>`, on the `$HOME/<CRI_SLURM_MOUNT_PATH>/<VOLUME CLAIM NAME>`.

## Job Results
Pod results will be stored in the NFS server, specifically in the path `/<NFS PATH>/<Sandobox ID>/<Container ID>`.  You can see the right path in the pod logs.

Data can be recovered by mounting the NFS path in your computer or
mounting the volume in a new pod (for example, busybox) and use `kubectl cp <new pod>:/<Mount Point>/<Sandboac ID><Container ID> <Local Path>`. 

### For example

* Slurm NFS configuration.

It is **important** to setup the mount point in Slurm as `$HOME/<CRI_SLURM_MOUNT_PATH>/<VOLUME CLAIM NAME>`, because the adapter will use it as working directory.
```
sudo mount <NFS server IP>:/mnt/storage/multicri-nfs /home/jorge/multi-cri/nfs-vol1
```

* K8s PV and PVC configuration:
  1. Create PV
  ```
  apiVersion: v1
  kind: PersistentVolume
  metadata:
    name: nfs-vol1
  spec:
    capacity:
      storage: 10Gi
    accessModes:
    - ReadWriteMany
    nfs:
      server: <CLUSTER IP>
      path: "/mnt/storage/multicri-nfs"
  ```
  2. Create PV
  ```
  apiVersion: v1
  kind: PersistentVolumeClaim
  apiVersion: v1
  metadata:
    name: nfs-vol1
  spec:
    accessModes:
    - ReadWriteMany
    storageClassName: ""
    resources:
      requests:
        storage: 10Gi
  ```

### Slurm simple batch job with volume
* Configure credentials through environment variables, they can be set, for example, by using K8s secrets.
* Use docker image repository.
* Use NFS data transfer. It mounts a Persistent Volume Claim called `nfs-vol1`
```
apiVersion: batch/v1
kind: Job
metadata:
  name: job-perl-slurm-vol-pod
spec:
  backoffLimit: 1
  template:
    metadata:
      labels:
        name: job-slurm-template
    spec:
      runtimeClassName: multicri
      containers:
      - name: job-slurm-container
        image: multicri/docker.perl.img:latest
        command: ["sleep", "60", "&&", "ls", "/"]
        env:
        - name: CLUSTER_USERNAME
          valueFrom:
            secretKeyRef:
              name: trujillo-secret
              key: username
        - name: CLUSTER_PASSWORD
          valueFrom:
            secretKeyRef:
              name: trujillo-secret
              key: password
        - name: CLUSTER_HOST
          valueFrom:
            secretKeyRef:
              name: trujillo-secret
              key: host
        - name: CLUSTER_PORT
          valueFrom:
            secretKeyRef:
              name: trujillo-secret
              key: port
        - name: CLUSTER_CONFIG
          valueFrom:
            secretKeyRef:
              name: trujillo-secret
              key: config
        - name: JOB_QUEUE
          valueFrom:
            secretKeyRef:
              name: trujillo-secret
              key: queue
        volumeMounts:
        # name must match the volume name below
          - name: my-pvc-nfs
            mountPath: "multicri"
      restartPolicy: Never
      nodeSelector:
        beta.kubernetes.io/arch: amd64
      volumes:
      - name: my-pvc-nfs
        persistentVolumeClaim:
          claimName: nfs-vol1
```

### MPI Slurm Job specification without NFS volume

```
apiVersion: batch/v1
kind: Job
metadata:
  name: job-perl-slurm-pod
spec:
  backoffLimit: 1
  template:
    metadata:
      labels:
        name: job-slurm-template
    spec:
      runtimeClassName: multicri
      containers:
      - name: job-slurm-container
        image: multicri/docker.perl:latest
        command: ["ls", "/"]
        env:
        - name: MPI_VERSION
          value: "1.10.2"
        - name: MPI_FLAGS
          value: "-np 2"
        - name: CLUSTER_USERNAME
          valueFrom:
            secretKeyRef:
              name: trujillo-secret
              key: username
        - name: CLUSTER_PASSWORD
          valueFrom:
            secretKeyRef:
              name: trujillo-secret
              key: password
        - name: CLUSTER_HOST
          valueFrom:
            secretKeyRef:
              name: trujillo-secret
              key: host
        - name: CLUSTER_PORT
          valueFrom:
            secretKeyRef:
              name: trujillo-secret
              key: port
        - name: CLUSTER_CONFIG
          valueFrom:
            secretKeyRef:
              name: trujillo-secret
              key: config
        - name: JOB_QUEUE
          valueFrom:
            secretKeyRef:
              name: trujillo-secret
              key: queue

      restartPolicy: Never
      nodeSelector:
        beta.kubernetes.io/arch: amd64

```

## Full setup
In the following, you can find the explanation of a full setup of this system.

First of all, we need to launch the docker CRI socket by using the kubelet command for both minikube and kubelet deployments. It can be done as a system service with the following configuration:
```
[Unit]
Description=dockershim for remote Multicri CRI
[Service]
ExecStart=/usr/bin/kubelet --experimental-dockershim --port 11250
Restart=always
StartLimitInterval=0
RestartSec=10

[Install]
RequiredBy=multi-cri.service
```
Second, we have to execute and configure multi-cri with the aforementioned parameters,
in addition to install Singularity 3.0. For instance, we can execute it as system service by using the following configuration:
```
[Unit]
Description=CRI Multicri
[Service]
Environment=CRI_SLURM_BUILD_IN_CLUSTER=true
ExecStart=/usr/local/bin/multi-cri -v 3 --socket-path /var/run/multi-cri.sock --remote-runtime-endpoints default:/var/run/dockershim.sock
Restart=always
StartLimitInterval=0
RestartSec=10
[Install]
WantedBy=kubelet.service # or localkube.service in case of using minikube
```
In case CNI network plugin raises a not found network configuration file exception, we can configure it by following the instructions of https://github.com/containernetworking/cni.

Third, we configure [RuntimeClass](https://kubernetes.io/docs/concepts/containers/runtime-class), it is supported by kubernetes 1.12.0 version.

Fourth, we need to create a runtimeClass instance for each runtime we want to use, except for the default runtimeClass.
```
# kubectl apply -f runtime_multicri.yaml

apiVersion: node.k8s.io/v1alpha1  # RuntimeClass is defined in the node.k8s.io API group
kind: RuntimeClass
metadata:
  name: multicri
  # The name the RuntimeClass will be referenced by
  # RuntimeClass is a non-namespaced resource
spec:
  runtimeHandler: multicri
```


Later, we configure Kubernetes to use multi-cri as remote container runtime. The following command shows how to do it:

`kubelet --container-runtime=remote --container-runtime-endpoint=/var/run/multi-cri.sock`.

In the case of using minikube, you can launch it by using your local machine as host and configuring the CRI parameters in this way:

`minikube start --kubernetes-version=v1.13.0 --vm-driver=none --extra-config=kubelet.container-runtime=remote --extra-config=kubelet.container-runtime-endpoint=/var/run/multi-cri.sock`



