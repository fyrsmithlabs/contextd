# Persona simulation Dockerfile for contextd installation testing
FROM ubuntu:22.04

ENV DEBIAN_FRONTEND=noninteractive
ENV USER=developer
ENV HOME=/home/${USER}

# Install essential developer tools
RUN apt-get update && apt-get install -y --no-install-recommends \
    curl wget git jq vim nano ca-certificates build-essential sudo \
    && rm -rf /var/lib/apt/lists/*

# Create developer user with sudo access
RUN useradd -m -s /bin/bash -G sudo ${USER} && \
    echo "${USER} ALL=(ALL) NOPASSWD:ALL" >> /etc/sudoers

USER ${USER}
WORKDIR ${HOME}

# Set up directory structure
RUN mkdir -p ${HOME}/.local/bin ${HOME}/.config/contextd ${HOME}/projects/my-app

ENV PATH="${HOME}/.local/bin:${PATH}"

# Copy binaries from build context
COPY --chown=${USER}:${USER} contextd ${HOME}/.local/bin/contextd
COPY --chown=${USER}:${USER} ctxd ${HOME}/.local/bin/ctxd
RUN chmod +x ${HOME}/.local/bin/contextd ${HOME}/.local/bin/ctxd

# Create sample project
RUN cd ${HOME}/projects/my-app && \
    git init && \
    git config user.email "test@example.com" && \
    git config user.name "Test User" && \
    echo "# My App" > README.md && \
    git add . && \
    git commit -m "Initial commit"

CMD ["/bin/bash"]
