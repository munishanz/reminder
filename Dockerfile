FROM golang:1.20.6-alpine

ENV DIR_HOME=/root
ENV DIR_DATA /data

WORKDIR ${DIR_DATA}

# copy required files
COPY main.go reminder/
COPY go.mod reminder/
COPY go.sum reminder/
COPY cmd reminder/cmd
COPY internal reminder/internal
COPY pkg reminder/pkg
COPY scripts reminder/scripts

# install the command
RUN cd reminder \
    && go install main.go

# rename the command
RUN cp ${GOPATH}/bin/main ${GOPATH}/bin/reminder

WORKDIR ${DIR_HOME}

CMD [ \
        "/bin/sh", "-c", \
        " \
        reminder \
        # while true; do echo \"Hit CTRL+C\"; sleep 1; done \
        " \
    ]
