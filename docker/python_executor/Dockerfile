# Python 执行器容器 - 隔离的 Python 执行环境
FROM python:3.11-slim

# 安装必要的系统依赖
RUN apt-get update && apt-get install -y \
    gcc \
    g++ \
    make \
    && rm -rf /var/lib/apt/lists/*

# 安装常用的 Python 科学计算库
RUN pip install --no-cache-dir \
    numpy \
    scipy \
    sympy \
    pandas \
    matplotlib \
    cryptography \
    requests

# 创建非 root 用户
RUN useradd -m -u 1000 pythonuser

# 设置工作目录
WORKDIR /app

# 创建执行脚本
RUN echo '#!/usr/bin/env python3\nimport sys\nimport json\ncode = sys.argv[1] if len(sys.argv) > 1 else ""\nexec(code)' > /app/execute.py && \
    chmod +x /app/execute.py && \
    chown pythonuser:pythonuser /app

# 切换到非 root 用户
USER pythonuser

# 限制资源使用（通过 ulimit）
# 注意：Docker 的资源限制应该在 docker-compose 或运行时配置

# 健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD python3 -c "import sys; sys.exit(0)" || exit 1

# 默认命令（可以通过 docker run 覆盖）
CMD ["python3", "-c", "import time; time.sleep(3600)"]

