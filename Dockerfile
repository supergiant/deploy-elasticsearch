FROM scratch
MAINTAINER Qbox Inc.
ADD deploy-elasticsearch deploy-elasticsearch
EXPOSE 8080
ENTRYPOINT ["/deploy-elasticsearch"]
