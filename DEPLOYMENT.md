# Deployment Guide

## ✅ การ Deploy สำเร็จแล้ว!

Go API Proxy ได้ถูก deploy ด้วย Docker Compose เรียบร้อยแล้ว

## 🚀 Quick Start Commands

### เริ่มต้นใช้งาน (Production)
```bash
docker-compose up -d
```

### ดู Status
```bash
docker-compose ps
```

### ดู Logs
```bash
docker-compose logs -f
```

### หยุดการทำงาน
```bash
docker-compose down
```

## 🔧 Environment Variables ที่ใช้งานอยู่

| Variable | Value | Description |
|----------|-------|-------------|
| `BACKEND_HOST` | `https://exp.co2e.cc` | Backend API URL |
| `PORT` | `80` | Server port |
| `WHITELIST_FILE` | `whitelist.json` | Token whitelist file |
| `HTTP_TIMEOUT` | `30` | HTTP timeout (seconds) |

## 📊 การทดสอบ

### Health Check
```bash
curl http://localhost/health
# Response: {"status":"healthy","service":"go-api-proxy"}
```

### Token API (Filtered)
```bash
curl http://localhost/api/v2/tokens
# Returns filtered tokens based on whitelist
```

## 🔄 การอัปเดต Whitelist

1. แก้ไขไฟล์ `whitelist.json`
2. ไม่ต้อง restart container (อัปเดตอัตโนมัติ)

## 🐳 Docker Commands

### Development Environment
```bash
# Start development (port 8080)
docker-compose -f docker-compose.dev.yml up -d

# View dev logs
docker-compose -f docker-compose.dev.yml logs -f

# Stop dev environment
docker-compose -f docker-compose.dev.yml down
```

### Production Environment
```bash
# Start production with advanced features
docker-compose -f docker-compose.prod.yml up -d

# View production logs
docker-compose -f docker-compose.prod.yml logs -f
```

### Using Makefile
```bash
make up        # Start production
make dev       # Start development
make logs      # View logs
make down      # Stop services
make clean     # Clean up everything
```

## 📈 Monitoring

### Container Status
```bash
docker-compose ps
```

### Resource Usage
```bash
docker stats go-api-proxy
```

### Health Check
```bash
# Manual health check
curl -f http://localhost/health

# Automated monitoring script
while true; do
  if curl -f http://localhost/health > /dev/null 2>&1; then
    echo "$(date): Service is healthy"
  else
    echo "$(date): Service is down"
  fi
  sleep 30
done
```

## 🔧 Troubleshooting

### Container ไม่ทำงาน
```bash
# ดู logs
docker-compose logs go-api-proxy

# Restart container
docker-compose restart go-api-proxy
```

### Port 80 ถูกใช้งานแล้ว
```bash
# เปลี่ยน port ใน docker-compose.yml
ports:
  - "8080:80"  # ใช้ port 8080 แทน
```

### Backend API ไม่ตอบสนอง
```bash
# ตรวจสอบ backend connectivity
curl https://exp.co2e.cc/api/v2/tokens

# เปลี่ยน backend ใน docker-compose.yml
environment:
  BACKEND_HOST: "https://your-backend-api.com"
```

## 🔒 Security Notes

- Container ทำงานด้วย non-root user (appuser:1001)
- Whitelist file เป็น read-only mount
- Health check endpoint พร้อมใช้งาน
- Resource limits ถูกกำหนดไว้แล้ว

## 📝 Next Steps

1. **Production Setup**: ใช้ `docker-compose.prod.yml` สำหรับ production
2. **SSL/TLS**: เพิ่ม nginx reverse proxy สำหรับ HTTPS
3. **Monitoring**: ตั้งค่า monitoring และ alerting
4. **Backup**: สำรองข้อมูล whitelist และ configuration
5. **CI/CD**: ตั้งค่า automated deployment pipeline

## 🎉 สำเร็จแล้ว!

Go API Proxy พร้อมใช้งานแล้วที่ http://localhost

- ✅ Docker Compose deployment
- ✅ Environment variables configuration  
- ✅ Health checks
- ✅ Token filtering
- ✅ Logging และ monitoring
- ✅ Graceful shutdown