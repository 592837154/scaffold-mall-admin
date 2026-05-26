---
title: Mall Admin
emoji: 🛒
colorFrom: blue
colorTo: green
sdk: docker
app_port: 7860
pinned: false
---

# Mall Admin

本项目包含两个运行容器：

- `frontend`: Ant Design Pro 前端，由 Nginx 提供静态资源。
- `goods-api`: Go + Gin + GORM 后端 API，直接连接 TiDB Cloud。

数据库统一使用 TiDB Cloud，本地和后续 Hugging Face 部署保持一致。

## 1. 首次准备 TiDB 密码

复制环境变量示例文件：

```bash
cd /d/learn/f
cp .env.example .env
```

如果你用 PowerShell：

```powershell
cd D:\learn\f
Copy-Item .env.example .env
```

然后打开 `D:\learn\f\.env`，把里面的值改成你在 TiDB Cloud 里生成的真实密码：

```env
MYSQL_PASSWORD=你的TiDB密码
```

注意：`.env` 已加入 `.gitignore`，不要把真实密码提交到 Git。

## 2. 本地启动

在项目根目录执行：

```bash
cd /d/learn/f
docker compose up -d --build
```

打开前端：

```text
http://localhost:8000/product-manage
```

后端接口：

```text
http://localhost:8080/api/goods/list?current=1&pageSize=10
```

## 3. 本地关闭

```bash
docker compose down
```

这只会关闭前端和后端容器，不会删除 TiDB Cloud 上的数据。

## 4. 查看状态

```bash
docker compose ps
```

## 5. 查看日志

全部服务：

```bash
docker compose logs -f
```

只看后端：

```bash
docker compose logs -f goods-api
```

只看前端：

```bash
docker compose logs -f frontend
```

## 6. 修改前端后重新构建

前端 Docker 镜像使用本地 `frontend/dist`。
改完前端代码后先构建前端：

```bash
cd /d/learn/f/frontend
pnpm build
cd ..
docker compose up -d --build
```

## 7. 修改后端后重新构建

后端 Docker 镜像使用本地 Linux 二进制文件 `backend/server`。
改完后端代码后先重新编译：

Git Bash：

```bash
cd /d/learn/f/backend
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o server main.go
cd ..
docker compose up -d --build
```

PowerShell：

```powershell
cd D:\learn\f\backend
$env:GOOS="linux"
$env:GOARCH="amd64"
$env:CGO_ENABLED="0"
go build -o server main.go
cd ..
docker compose up -d --build
```

## 8. TiDB Cloud 连接配置

当前 `docker-compose.yml` 已固定使用你的 TiDB Cloud 实例：

```text
MYSQL_HOST=gateway01.ap-northeast-1.prod.aws.tidbcloud.com
MYSQL_PORT=4000
MYSQL_USER=3d1HaJiKd8td5iE.root
MYSQL_DATABASE=mall_admin
MYSQL_TLS=skip-verify
```

`MYSQL_PASSWORD` 从本地 `.env` 读取。

说明：TiDB Cloud 要求 TLS。当前容器为了保持镜像简单，使用 `skip-verify`，仍然走 TLS 加密，但不校验证书链。正式生产环境建议改成 CA 证书校验。

## 9. 部署到 Hugging Face Spaces

Hugging Face Spaces 不能直接使用本项目的 `docker-compose.yml`。
线上使用根目录的 `Dockerfile`，把前端 Nginx 和后端 Go 服务放进同一个容器。

在 Hugging Face 创建或打开你的 Space：

1. 进入 Space 页面。
2. 点击 `Settings`。
3. 找到 `Repository secrets`。
4. 新增这些环境变量：

```text
MYSQL_HOST=gateway01.ap-northeast-1.prod.aws.tidbcloud.com
MYSQL_PORT=4000
MYSQL_USER=3d1HaJiKd8td5iE.root
MYSQL_PASSWORD=你的TiDB密码
MYSQL_DATABASE=mall_admin
MYSQL_TLS=skip-verify
```

然后把代码推送到 Space 仓库。

如果 Space 仓库是空的，先添加远程地址：

```bash
git remote add hf https://huggingface.co/spaces/你的用户名/你的Space名
```

推送：

```bash
git add .
git commit -m "Deploy mall admin to Hugging Face"
git push hf main
```

如果你的本地分支叫 `master`：

```bash
git push hf master:main
```

Hugging Face 构建时会读取根目录 `Dockerfile`，容器启动后访问 Space 页面即可打开后台系统。
