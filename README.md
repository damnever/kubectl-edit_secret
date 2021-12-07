A [kubectl plugin](https://kubernetes.io/docs/tasks/extend-kubectl/kubectl-plugins/) to [edit Secret via `stringData`](https://kubernetes.io/docs/concepts/configuration/secret/#overview-of-secrets).


### Installation

1. Install via Golang:
    ```bash
    go install github.com/damnever/kubectl-edit_secret@latest
    ```
2. Build manually:
    ```bash
    git clone https://github.com/damnever/kubectl-edit_secret.git
    go build -o kubectl-edit_secret && mv kubectl-edit_secret /usr/local/bin/
    ```

### Usage

```bash
kubectl [-n NAMESPACE] edit-secret SECRET_NAME

kubectl edit-secret --help
```
