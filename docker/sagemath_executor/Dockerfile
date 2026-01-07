# SageMath 执行器容器 - 隔离的 SageMath 执行环境
FROM sagemath/sagemath:latest

# 创建非 root 用户
RUN useradd -m -u 1000 sageuser

# 设置工作目录
WORKDIR /app

# 创建执行脚本
RUN echo '#!/usr/bin/env sage\nimport sys\ncode = sys.argv[1] if len(sys.argv) > 1 else ""\nexec(code)' > /app/execute.py && \
    chmod +x /app/execute.py && \
    chown sageuser:sageuser /app

# 切换到非 root 用户
USER sageuser

# 健康检查
HEALTHCHECK --interval=30s --timeout=10s --start-period=10s --retries=3 \
    CMD sage -c "1+1" || exit 1

# 默认命令（可以通过 docker run 覆盖）
CMD ["sage", "-c", "import time; time.sleep(3600)"]

