# Hướng dẫn kiểm thử thủ công WorkTrack

> Dành cho người **chưa biết gì** về dự án, muốn tự kiểm thử bằng tay các chức năng đã có:
> scaffold, đăng nhập (CF-01), cơ cấu tổ chức và tài khoản (CF-02), dự án và bảng Kanban (CF-03),
> công việc cha (CF-06), Kanban kéo thả và WIP (CF-04, CF-05).
>
> Mọi lệnh, đoạn curl và câu SQL dưới đây đã được chạy thử trên máy dev macOS (không Docker, theo
> `docs/decisions.md` B12) ngày 2026-11-21. Ngoại lệ được ghi rõ ở §7.4.
> Mật khẩu viết dạng `<MK_ADMIN>`, `<MK_MANAGER>`… là **chỗ để điền**. Mật khẩu thật do `make seed` in ra
> và **đổi mỗi lần chạy**.

Mục lục:
1. [Chuẩn bị môi trường (không Docker)](#1-chuẩn-bị-môi-trường-không-docker)
2. [Đăng nhập, đăng xuất, làm mới phiên, khóa tài khoản](#2-đăng-nhập-đăng-xuất-làm-mới-phiên-khóa-tài-khoản)
3. [Admin tạo tài khoản → email → kích hoạt](#3-admin-tạo-tài-khoản--email-trong-mailpit--kích-hoạt)
4. [Kiểm thử từng chức năng theo vai trò](#4-kiểm-thử-từng-chức-năng-theo-vai-trò)
5. [Xem CSDL bằng psql và pgAdmin](#5-xem-csdl-bằng-psql-và-pgadmin)
6. [Thử API bằng curl](#6-thử-api-bằng-curl)
7. [Đặt lại dữ liệu dev an toàn](#7-đặt-lại-dữ-liệu-dev-an-toàn)
8. [Checklist CF-01, CF-02, CF-03, CF-06](#8-checklist-kiểm-thử)

---

## 1. Chuẩn bị môi trường (không Docker)

### 1.1. Công cụ cần có
Cài theo `docs/setup.md`. Bảng dưới là các lệnh kiểm tra nhanh:

| Công cụ | Kiểm tra |
|---|---|
| Go | `go version` |
| Node.js + npm | `node -v && npm -v` |
| PostgreSQL 16 (Homebrew) | `pg_isready -h localhost -p 5432` |
| Mailpit (Homebrew) | `curl -s http://localhost:8025/api/v1/info \| jq .Version` |
| goose | `goose --version` |
| jq, curl, openssl | `jq --version; curl --version \| head -1; openssl version` |

### 1.2. Kiểm tra PostgreSQL và Mailpit đang chạy
1. Xem dịch vụ:
   ```bash
   brew services list | grep -Ei 'postgres|mailpit'
   ```
   Kết quả mong đợi: `postgresql@16` và `mailpit` ở trạng thái `started`.
2. Nếu chưa chạy:
   ```bash
   brew services start postgresql@16
   brew services start mailpit
   ```
3. Kiểm tra cổng:
   ```bash
   pg_isready -h localhost -p 5432        # "localhost:5432 - accepting connections"
   nc -z localhost 1025 && echo "SMTP Mailpit OK"
   nc -z localhost 8025 && echo "Web Mailpit OK"
   ```
4. Lần đầu trên máy mới (chưa có CSDL), tạo user và CSDL `worktrack`:
   <!-- verify:skip (CSDL đã có trên máy kiểm thử) -->
   ```bash
   createuser -h localhost -s worktrack
   psql -h localhost -d postgres -c "ALTER USER worktrack WITH PASSWORD 'worktrack';"
   createdb -h localhost -O worktrack worktrack
   ```

### 1.3. File `.env` ở gốc repo
`scripts/dev-local.sh` và `scripts/seed.sh` đọc `.env` ở gốc repo. Nếu chưa có file này, sao chép từ
`.env.example` rồi sửa:
<!-- verify:skip (không đọc/ghi .env trong phiên tự động) -->
```bash
cp .env.example .env
```

Danh sách **đầy đủ** biến mà code đọc. Nguồn: `backend/cmd/api/config.go` và `frontend/src/shared/api/`.

| Biến | Bắt buộc | Mặc định nếu bỏ trống | Giá trị mẫu cho dev | Ghi chú |
|---|---|---|---|---|
| `DATABASE_URL` | **Có** | — | `postgres://worktrack:worktrack@localhost:5432/worktrack?sslmode=disable` | Thiếu → API dừng với `DATABASE_URL is required` |
| `JWT_SECRET` | **Có** | — | kết quả của `openssl rand -hex 32` (64 ký tự) | Phải ≥ 32 byte, nếu không API dừng với `JWT_SECRET is required and must be at least 32 bytes` |
| `APP_ENV` | Không | `development` | `development` | |
| `HTTP_ADDR` | Không | `:8080` | `:8080` | Địa chỉ API lắng nghe |
| `CORS_ORIGIN` | Không | `http://localhost:5173` | `http://localhost:5173` | Một origin duy nhất, phải trùng URL web |
| `ACCESS_TOKEN_TTL` | Không | `15m` | `15m` | Thời hạn access token. Sai định dạng → `ACCESS_TOKEN_TTL must be a positive duration, e.g. 15m` |
| `APP_BASE_URL` | Không | `http://localhost:5173` | `http://localhost:5173` | Dùng để dựng link kích hoạt trong email |
| `SMTP_HOST` | Không | `localhost` | `localhost` | Mailpit |
| `SMTP_PORT` | Không | `1025` | `1025` | Sai → `SMTP_PORT must be a valid port` |
| `SMTP_FROM` | Không | `WorkTrack <no-reply@worktrack.local>` | như mặc định | |
| `SMTP_USERNAME` | Không | rỗng | rỗng | Mailpit không cần |
| `SMTP_PASSWORD` | Không | rỗng | rỗng | Mailpit không cần |
| `VITE_API_BASE_URL` | Không | `http://localhost:8080` | `http://localhost:8080` | Frontend dùng để gọi API |

Sinh `JWT_SECRET`:
```bash
openssl rand -hex 32
```
Dán kết quả vào dòng `JWT_SECRET=` trong `.env`. Không commit `.env`.

> Lưu ý: `make migrate-up` **không đọc `.env`**. Nó dùng `DATABASE_URL` của môi trường, nếu không có thì
> dùng mặc định trong Makefile (cùng giá trị mẫu ở trên). Nếu `.env` của bạn dùng URL khác, chạy
> `make migrate-up DATABASE_URL='<url của bạn>'`.

### 1.4. Tạo bảng và dữ liệu mẫu
1. Áp dụng migration:
   ```bash
   make migrate-up
   ```
   Lần đầu sẽ thấy các dòng `OK   2026...sql`. Lần sau: `goose: no migrations to run. current version: ...`.
2. Kiểm tra trạng thái migration:
   ```bash
   cd backend && goose -dir db/migrations postgres "postgres://worktrack:worktrack@localhost:5432/worktrack?sslmode=disable" status; cd ..
   ```
   Mọi dòng phải có ngày ở cột `Applied At` (không có `Pending`).
3. Tạo dữ liệu mẫu:
   ```bash
   make seed
   ```
   Kết quả mẫu (mật khẩu thật khác, **đổi mỗi lần chạy**):
   ```text
   Tài khoản mẫu (chỉ dùng cho môi trường dev):
     admin    admin@worktrack.local      <MK_ADMIN>
     ceo      ceo@worktrack.local        <MK_CEO>
     manager  manager@worktrack.local    <MK_MANAGER>
     member   member@worktrack.local     <MK_MEMBER>
   Cơ cấu tổ chức: 6 phòng ban, 5 chức danh, 8 kỹ năng.
   Dự án mẫu: 2 dự án (mới tạo 0).
   Công việc mẫu: 4 nhãn, 6 công việc (mới tạo 0).
   ```
   **Ghi lại 4 mật khẩu.** Seed chạy lại được nhiều lần. Phòng ban, dự án và công việc mẫu không bị
   tạo trùng; chỉ 4 mật khẩu được đặt lại và các tài khoản mẫu được mở khóa.

### 1.5. Chạy app
<!-- verify:skip (chạy liên tục; đã chạy nền khi kiểm thử) -->
```bash
make dev-local
```
Lệnh này chạy cùng lúc API (`go run ./cmd/api`) và web (`npm run dev`). Bấm `Ctrl+C` để dừng cả hai.
Mở một terminal khác để làm các bước tiếp theo.

| Thành phần | URL |
|---|---|
| Web | http://localhost:5173 |
| API | http://localhost:8080/api/v1 |
| Kiểm tra sống | http://localhost:8080/healthz → `{"status":"ok"}` |
| Kiểm tra sẵn sàng (có CSDL) | http://localhost:8080/readyz → `{"status":"ready"}` |
| Mailpit (hộp thư dev) | http://localhost:8025 |

Kiểm tra:
```bash
curl -s http://localhost:8080/healthz
curl -s http://localhost:8080/readyz
curl -s -o /dev/null -w '%{http_code}\n' http://localhost:5173    # 200
```

### 1.6. Lỗi thường gặp khi khởi động
| Triệu chứng | Nguyên nhân | Cách xử lý |
|---|---|---|
| `DATABASE_URL is required` | `.env` thiếu biến hoặc chưa có `.env` | Thêm biến theo §1.3 |
| `JWT_SECRET is required and must be at least 32 bytes` | Thiếu hoặc quá ngắn | `openssl rand -hex 32`, dán vào `.env` |
| `SMTP_PORT must be a valid port` / `ACCESS_TOKEN_TTL must be a positive duration…` | Giá trị sai định dạng | Sửa thành `1025` / `15m` |
| `/readyz` không trả `ready`, log có cảnh báo kết nối CSDL | PostgreSQL chưa chạy hoặc sai URL | `brew services start postgresql@16`; kiểm `pg_isready`; so `DATABASE_URL` |
| `bind: address already in use` (8080) hoặc Vite báo cổng 5173 bận | Tiến trình cũ còn chạy | `lsof -i :8080` (hoặc `:5173`), dừng tiến trình đó rồi chạy lại |
| API báo lỗi 500, log có `relation "..." does not exist` | Chưa chạy migration | `make migrate-up` |
| Đăng nhập báo sai mật khẩu với mật khẩu cũ | Đã chạy `make seed` lại → mật khẩu đổi | Dùng mật khẩu của lần seed gần nhất |
| Không thấy email kích hoạt | Mailpit chưa chạy (API vẫn tạo tài khoản, `emailSent:false`) | `brew services start mailpit`, rồi gửi lại email (§3, bước 7) |
| Trình duyệt báo lỗi CORS | `CORS_ORIGIN` khác URL web | Đặt `CORS_ORIGIN=http://localhost:5173` |
| `429` `RATE_LIMITED` khi đăng nhập | Quá 10 lần đăng nhập/kích hoạt liên tiếp từ cùng IP | Chờ vài giây (hồi 1 lượt mỗi 6 giây; xem header `Retry-After`) |

---

## 2. Đăng nhập, đăng xuất, làm mới phiên, khóa tài khoản

### 2.1. Trên giao diện
1. Mở http://localhost:5173. Chưa đăng nhập thì app tự chuyển tới `/login`.
2. Đăng nhập lần lượt bằng 4 tài khoản mẫu (email ở §1.4, mật khẩu do seed in ra). Kết quả mong đợi:

   | Tài khoản | Menu thấy được |
   |---|---|
   | `admin@worktrack.local` | Trang chủ, Quản trị (`/admin/users`: Tài khoản, Phòng ban, Chức danh, Kỹ năng, Nhãn, Cấu hình) — **không** có Dự án |
   | `ceo@worktrack.local` | Trang chủ, Dự án (chỉ xem), Tổng quan công ty, Quản trị |
   | `manager@worktrack.local` | Trang chủ, Dự án (có nút tạo dự án), Báo cáo |
   | `member@worktrack.local` | Trang chủ, Dự án (chỉ dự án mình tham gia), Công việc của tôi |

3. Đăng xuất bằng menu tài khoản ở góc trên. App quay về `/login`. Bấm Back của trình duyệt không vào lại được trang trong.
4. Làm mới phiên: đăng nhập, bấm F5. Bạn vẫn ở trạng thái đăng nhập vì app gọi `POST /auth/refresh` bằng
   cookie `wt_refresh` (HttpOnly, 7 ngày).
5. Ca lỗi nên thử:
   - Email sai định dạng hoặc để trống → form báo lỗi ngay, không gửi request.
   - Mật khẩu sai → thông báo chung "Email hoặc mật khẩu không đúng" (không tiết lộ email có tồn tại hay không).

### 2.2. Bằng curl (cũng là cách chuẩn bị token cho §4 và §6)
1. Khai báo biến và hai hàm tiện ích. Chạy trong **bash hoặc zsh** và giữ terminal này cho các bước sau:
   ```bash
   export API=http://localhost:8080/api/v1
   # login <vai trò> '<mật khẩu>' — lưu access token vào /tmp/tok-<vai trò>, cookie refresh vào /tmp/wt-<vai trò>.jar
   login() {
     jq -n --arg e "$1@worktrack.local" --arg p "$2" '{email:$e,password:$p}' |
       curl -s -c /tmp/wt-$1.jar -X POST $API/auth/login -H 'Content-Type: application/json' -d @- |
       jq -r .accessToken > /tmp/tok-$1
     echo "$1: $(head -c 20 /tmp/tok-$1)…"
   }
   # H <vai trò> — in header Authorization tương ứng
   H() { echo "Authorization: Bearer $(cat /tmp/tok-$1)"; }
   ```
2. Đăng nhập 4 vai trò (thay mật khẩu thật):
   ```bash
   login admin   '<MK_ADMIN>'
   login ceo     '<MK_CEO>'
   login manager '<MK_MANAGER>'
   login member  '<MK_MEMBER>'
   ```
   Mỗi dòng phải in ra đầu token dạng `eyJhbGciOi…`. Nếu in `null…`: sai mật khẩu, tài khoản bị khóa, hoặc
   bị giới hạn tần suất (429 — chờ vài giây rồi chạy lại).
3. Xem mình là ai:
   ```bash
   curl -s $API/auth/me -H "$(H manager)" | jq '{email, role, status, permissions}'
   ```
   Mong đợi với manager: `"role": "manager"`, `permissions` gồm `projects.create`, `projects.view_own`, `dashboard.team.view`.
4. Không có token:
   ```bash
   curl -s $API/auth/me | jq .
   ```
   Mong đợi: HTTP 401, `{"error":{"code":"UNAUTHENTICATED","message":"Phiên đăng nhập đã hết hạn, vui lòng đăng nhập lại"}}`.
5. Làm mới phiên. Cookie refresh **xoay vòng**: mỗi lần refresh, server cấp cookie mới và thu hồi cookie cũ.
   ```bash
   curl -s -b /tmp/wt-manager.jar -c /tmp/wt-manager.jar -X POST $API/auth/refresh | jq -r .accessToken > /tmp/tok-manager
   head -c 20 /tmp/tok-manager; echo
   ```
   Mong đợi: có token mới. Luôn dùng cả `-b` và `-c` để cookie jar luôn giữ cookie mới nhất.
   Dùng lại cookie **cũ** đã bị xoay → 401 `UNAUTHENTICATED`, và server thu hồi luôn cả họ token. Đây là cơ chế
   phát hiện token bị đánh cắp.
6. Đăng xuất:
   ```bash
   curl -s -o /dev/null -w '%{http_code}\n' -b /tmp/wt-manager.jar -c /tmp/wt-manager.jar -X POST $API/auth/logout   # 204
   curl -s -o /dev/null -w '%{http_code}\n' -b /tmp/wt-manager.jar -X POST $API/auth/refresh                          # 401
   ```
   Access token cũ vẫn dùng được tới khi hết hạn (15 phút). Đăng nhập lại manager trước khi sang bước sau:
   ```bash
   login manager '<MK_MANAGER>'
   ```

### 2.3. Khóa tài khoản sau nhiều lần sai mật khẩu
Quy tắc: sai **5 lần liên tiếp** → khóa 15 phút. Ngay lần sai thứ 5, server đã trả 423.
1. Nhập sai 5 lần cho member:
   ```bash
   for i in 1 2 3 4 5; do
     curl -s -X POST $API/auth/login -H 'Content-Type: application/json' \
       -d '{"email":"member@worktrack.local","password":"sai-mat-khau"}' | jq -c .error
   done
   ```
   Mong đợi: 4 dòng `AUTH_INVALID_CREDENTIALS`, dòng thứ 5 là
   `{"code":"AUTH_ACCOUNT_LOCKED","message":"Tài khoản tạm khóa 15 phút do nhập sai mật khẩu nhiều lần"}`.
   Lúc này đăng nhập **đúng** mật khẩu cũng bị từ chối.
2. Xem trong CSDL:
   ```sql
   SELECT email, failed_login_count, locked_until FROM app_user WHERE email = 'member@worktrack.local';
   ```
   `locked_until` ≈ bây giờ + 15 phút; `failed_login_count` = 0 (bộ đếm tự đặt lại khi tài khoản bị khóa — đã quan sát khi chạy thử).
3. Mở khóa (chọn một cách):
   - chờ 15 phút; hoặc
   - chạy `make seed` (đặt lại mật khẩu **và** mở khóa 4 tài khoản mẫu); hoặc
   - chạy SQL:
     ```sql
     UPDATE app_user SET failed_login_count = 0, locked_until = NULL WHERE email = 'member@worktrack.local';
     ```
   Sau đó `login member '<MK_MEMBER>'` lại.
4. Giới hạn tần suất: đăng nhập và kích hoạt dùng chung giới hạn 10 lượt, hồi 1 lượt mỗi 6 giây, tính theo
   IP. Bấm liên tục sẽ nhận 429 `RATE_LIMITED` kèm header `Retry-After`. Refresh có giới hạn riêng: 30 lượt,
   hồi 1 lượt mỗi 2 giây.

---

## 3. Admin tạo tài khoản → email trong Mailpit → kích hoạt

### 3.1. Trên giao diện
1. Đăng nhập `admin@worktrack.local` → **Quản trị** → tab **Tài khoản** → **Thêm tài khoản**.
2. Nhập họ tên, email mới (ví dụ `nv.thu1@worktrack.local`), vai trò **Member**, phòng ban, chức danh → Lưu.
   Mong đợi: tài khoản mới ở trạng thái **Chờ kích hoạt**.
3. Mở http://localhost:8025. Có email tiêu đề **"Kích hoạt tài khoản WorkTrack"** gửi tới email vừa nhập.
   Nội dung có link `http://localhost:5173/activate?token=…`, hiệu lực 72 giờ, chỉ dùng một lần.
4. Bấm link (nên mở trong cửa sổ ẩn danh để không lẫn phiên Admin). Trang hiển thị email và họ tên → đặt
   mật khẩu → xác nhận.
5. Thông báo kích hoạt thành công → tới trang đăng nhập. Đăng nhập bằng mật khẩu vừa đặt.
6. Ca lỗi nên thử:
   - Mở lại link lần hai → báo link không hợp lệ hoặc đã dùng.
   - Mật khẩu yếu (`abcdefgh`, `abc1`) → báo mật khẩu chưa đủ mạnh.
   - Trước khi kích hoạt, đăng nhập bằng email mới → "Email hoặc mật khẩu không đúng".
   - Admin tạo tài khoản trùng email → "Email đã tồn tại".
   - Admin không chọn được vai trò CEO (chỉ CEO tạo được CEO).
7. Gửi lại email kích hoạt: trong danh sách tài khoản, chọn tài khoản đang chờ → **Gửi lại email kích hoạt**.
   Làm vậy với tài khoản đã kích hoạt → "Tài khoản đã được kích hoạt".

### 3.2. Bằng curl (cần biến và token ở §2.2)
1. Admin tạo tài khoản. Email có hậu tố thời gian để chạy lại không bị trùng:
   ```bash
   NEW_EMAIL="nv.$(date +%s)@worktrack.local"
   jq -n --arg e "$NEW_EMAIL" '{email:$e, fullName:"Nhân viên Thử", role:"member"}' |
     curl -s -X POST $API/users -H "$(H admin)" -H 'Content-Type: application/json' -d @- |
     tee /tmp/newuser.json | jq '{status: .user.status, emailSent}'
   NEW_ID=$(jq -r .user.id /tmp/newuser.json)
   ```
   Mong đợi: `{"status":"pending","emailSent":true}`.
2. Lấy token từ Mailpit:
   ```bash
   MSG=$(curl -s http://localhost:8025/api/v1/messages |
     jq -r --arg e "$NEW_EMAIL" '[.messages[] | select(.To[0].Address == $e)][0].ID')
   curl -s http://localhost:8025/api/v1/message/$MSG | jq -r .Subject
   TOKEN=$(curl -s http://localhost:8025/api/v1/message/$MSG | jq -r .Text |
     grep -o 'token=[A-Za-z0-9_-]*' | head -1 | cut -d= -f2 | tr -d '\r\n')
   echo "${#TOKEN} ký tự"
   ```
   > Nội dung email dùng xuống dòng CRLF. Nếu tự cắt token bằng cách khác, nhớ bỏ ký tự `\r` (`tr -d '\r\n'`),
   > nếu không API sẽ trả `INVALID_JSON` hoặc báo token không hợp lệ.
3. Kiểm tra token (trang `/activate` gọi API này để hiện email và họ tên):
   ```bash
   jq -n --arg t "$TOKEN" '{token:$t}' |
     curl -s -X POST $API/auth/activation/verify -H 'Content-Type: application/json' -d @- | jq .
   ```
   Mong đợi: `{"email": "<NEW_EMAIL>", "fullName": "Nhân viên Thử"}`.
4. Ca lỗi — mật khẩu yếu:
   ```bash
   jq -n --arg t "$TOKEN" '{token:$t, password:"abcdefgh"}' |
     curl -s -X POST $API/auth/activation -H 'Content-Type: application/json' -d @- | jq -c .error
   ```
   Mong đợi: 422 `PASSWORD_TOO_WEAK`.
5. Kích hoạt với mật khẩu hợp lệ:
   ```bash
   jq -n --arg t "$TOKEN" '{token:$t, password:"Matkhau123"}' |
     curl -s -o /dev/null -w '%{http_code}\n' -X POST $API/auth/activation -H 'Content-Type: application/json' -d @-
   ```
   Mong đợi: `204`. API **không** tạo phiên đăng nhập; bạn phải đăng nhập riêng.
6. Dùng lại token → `ACTIVATION_TOKEN_INVALID`:
   ```bash
   jq -n --arg t "$TOKEN" '{token:$t, password:"Matkhau123"}' |
     curl -s -X POST $API/auth/activation -H 'Content-Type: application/json' -d @- | jq -c .error
   ```
7. Đăng nhập tài khoản mới:
   ```bash
   jq -n --arg e "$NEW_EMAIL" '{email:$e, password:"Matkhau123"}' |
     curl -s -X POST $API/auth/login -H 'Content-Type: application/json' -d @- | jq '{role: .user.role, status: .user.status}'
   ```
   Mong đợi: `{"role":"member","status":"active"}`.
8. Gửi lại email cho tài khoản đã kích hoạt → `USER_ALREADY_ACTIVE`:
   ```bash
   curl -s -X POST $API/users/$NEW_ID/activation-email -H "$(H admin)" | jq -c .error
   ```
9. Xem trong CSDL. Sau khi kích hoạt, `co_token` = `false` và `status` = `active`:
   ```sql
   SELECT email, status, activation_token_hash IS NOT NULL AS co_token, activation_expires_at
   FROM app_user ORDER BY created_at DESC LIMIT 5;
   ```

---

## 4. Kiểm thử từng chức năng theo vai trò

Ma trận quyền tóm tắt (server kiểm tra, không chỉ ẩn nút):

| Chức năng | Admin | CEO | Manager | Member |
|---|---|---|---|---|
| Tài khoản (tạo/sửa/kỹ năng) | ✅ (không tạo CEO) | ✅ | ❌ 403 | ❌ 403 |
| Phòng ban, chức danh, kỹ năng | ✅ | ✅ | ❌ 403 | ❌ 403 |
| Xem danh sách dự án | ❌ 403 | ✅ toàn bộ, chỉ xem | ✅ dự án mình phụ trách hoặc tham gia | ✅ dự án mình tham gia |
| Tạo/sửa dự án, thành viên, cột, luật chuyển | ❌ | ❌ | ✅ dự án mình phụ trách | ❌ |
| Tạo/sửa công việc cha | ❌ | ❌ | ✅ dự án mình phụ trách | ❌ 403 |
| Nhận việc (claim) | ❌ | ❌ | ❌ 403 | ✅ việc chưa giao ở nhóm Cần làm |
| Công việc của tôi, WIP | ❌ 403 | ❌ 403 | ✅ | ✅ |
| Nhãn, cấu hình giờ/Story Point | ✅ ghi | ✅ ghi | đọc | đọc |

Mọi lỗi API có dạng `{"error":{"code":"…","message":"…"}}`. Câu `message` là câu tiếng Việt mà UI hiển thị.

Các bước dưới cần biến `API`, hàm `H`, `login` và 4 token ở §2.2. Câu SQL chạy trong `psql` (§5).

### 4.1. CF-02 — Cơ cấu tổ chức (Admin, CEO)

**Trên giao diện** (Admin → Quản trị):
1. Tab **Phòng ban**: thêm phòng ban gốc, thêm phòng ban con, đổi tên, chuyển phòng ban cha, xóa.
2. Tab **Chức danh**, **Kỹ năng**: thêm, đổi tên, xóa. Cột "Số nhân sự" cho biết đang được dùng bao nhiêu.
3. Tab **Tài khoản**: lọc theo vai trò, trạng thái, phòng ban, tìm theo tên/email; sửa họ tên, vai trò,
   phòng ban, chức danh; gán kỹ năng kèm cấp độ (Beginner, Intermediate, Senior).
4. Đăng nhập Manager hoặc Member, gõ thẳng http://localhost:5173/admin/users → "Bạn không có quyền xem trang này".

**Bằng curl:**
1. Danh sách phòng ban (Admin được, Manager bị 403):
   ```bash
   curl -s $API/departments -H "$(H admin)" | jq -c '.items[] | {name, parentId, memberCount}'
   curl -s $API/departments -H "$(H manager)" | jq -c .error
   ```
2. Tạo phòng ban gốc và phòng ban con, rồi thử tạo vòng lặp (chuyển cha vào con):
   ```bash
   SUF=$(date +%s)
   D1=$(jq -n --arg n "Khối thử $SUF" '{name:$n}' | curl -s -X POST $API/departments -H "$(H admin)" -H 'Content-Type: application/json' -d @- | jq -r .id)
   D2=$(jq -n --arg n "Nhóm con $SUF" --arg p "$D1" '{name:$n, parentId:$p}' | curl -s -X POST $API/departments -H "$(H admin)" -H 'Content-Type: application/json' -d @- | jq -r .id)
   jq -n --arg n "Khối thử $SUF" --arg p "$D2" '{name:$n, parentId:$p}' |
     curl -s -X PUT $API/departments/$D1 -H "$(H admin)" -H 'Content-Type: application/json' -d @- | jq -c .error
   ```
   Mong đợi: `DEPARTMENT_CYCLE` "Không thể chuyển phòng ban vào chính nó hoặc phòng ban con của nó".
3. Trùng tên trong cùng phòng ban cha:
   ```bash
   jq -n --arg n "Nhóm con $SUF" --arg p "$D1" '{name:$n, parentId:$p}' |
     curl -s -X POST $API/departments -H "$(H admin)" -H 'Content-Type: application/json' -d @- | jq -c .error
   ```
   Mong đợi: `DEPARTMENT_NAME_TAKEN`. Cùng tên nhưng khác phòng ban cha thì được phép.
4. Xóa phòng ban còn phòng ban con → `DEPARTMENT_HAS_CHILDREN`. Xóa con trước rồi xóa cha → 204:
   ```bash
   curl -s -X DELETE $API/departments/$D1 -H "$(H admin)" | jq -c .error
   curl -s -o /dev/null -w '%{http_code}\n' -X DELETE $API/departments/$D2 -H "$(H admin)"
   curl -s -o /dev/null -w '%{http_code}\n' -X DELETE $API/departments/$D1 -H "$(H admin)"
   ```
   Xóa phòng ban còn nhân sự → `DEPARTMENT_IN_USE`.
5. Kỹ năng trùng tên (không phân biệt hoa thường), kỹ năng/chức danh đang dùng:
   ```bash
   curl -s -X POST $API/skills -H "$(H admin)" -H 'Content-Type: application/json' -d '{"name":"go"}' | jq -c .error
   curl -s $API/job-titles -H "$(H admin)" | jq -c '.items[] | {name, memberCount}'
   ```
   Mong đợi: `SKILL_NAME_TAKEN` (seed đã có "Go"). Xóa mục có `memberCount > 0` → `JOB_TITLE_IN_USE` / `SKILL_IN_USE`.
6. Tài khoản — danh sách, lọc, phân quyền:
   ```bash
   curl -s "$API/users?role=member&status=active&pageSize=5" -H "$(H admin)" | jq '{total, emails: [.items[].email]}'
   curl -s -G $API/users --data-urlencode 'search=Nhân viên' -H "$(H ceo)" | jq .total
   curl -s $API/users -H "$(H manager)" | jq -c .error
   ```
   Mong đợi: hai lệnh đầu trả dữ liệu; Manager → 403 `FORBIDDEN`.
7. Ca lỗi khi tạo tài khoản:
   ```bash
   curl -s -X POST $API/users -H "$(H admin)" -H 'Content-Type: application/json' -d '{"email":"admin@worktrack.local","fullName":"Trùng","role":"member"}' | jq -c .error
   curl -s -X POST $API/users -H "$(H admin)" -H 'Content-Type: application/json' -d '{"email":"khong-phai-email","fullName":"X","role":"member"}' | jq -c .error
   curl -s -X POST $API/users -H "$(H admin)" -H 'Content-Type: application/json' -d '{"email":"ceo2@worktrack.local","fullName":"X","role":"ceo"}' | jq -c .error
   ```
   Mong đợi theo thứ tự: `USER_EMAIL_TAKEN`, `VALIDATION_FAILED`, `FORBIDDEN` (Admin không tạo được CEO).
8. Sửa tài khoản (dùng `NEW_ID` ở §3.2). Admin không tự đổi vai trò của mình:
   ```bash
   jq -n '{fullName:"Nhân viên Thử (sửa)", role:"member"}' |
     curl -s -X PUT $API/users/$NEW_ID -H "$(H admin)" -H 'Content-Type: application/json' -d @- | jq '{fullName, role}'
   ADMIN_ID=$(curl -s $API/auth/me -H "$(H admin)" | jq -r .id)
   jq -n '{fullName:"Quản trị viên", role:"member"}' |
     curl -s -X PUT $API/users/$ADMIN_ID -H "$(H admin)" -H 'Content-Type: application/json' -d @- | jq -c .error
   ```
   Mong đợi: lệnh đầu trả họ tên mới; lệnh sau `USER_CANNOT_CHANGE_OWN_ROLE`. Body có trường lạ (ví dụ `email`)
   → `INVALID_JSON`. Email không đổi được.
9. Gán kỹ năng:
   ```bash
   SKILL=$(curl -s $API/skills -H "$(H admin)" | jq -r '.items[0].id')
   jq -n --arg s "$SKILL" '{skills:[{skillId:$s, level:"senior"}]}' |
     curl -s -X PUT $API/users/$NEW_ID/skills -H "$(H admin)" -H 'Content-Type: application/json' -d @- | jq -c .
   jq -n --arg s "$SKILL" '{skills:[{skillId:$s, level:"expert"}]}' |
     curl -s -X PUT $API/users/$NEW_ID/skills -H "$(H admin)" -H 'Content-Type: application/json' -d @- | jq -c .error
   ```
   Mong đợi: `{"skills":[{"skillId":…,"name":…,"level":"senior"}]}`, rồi `SKILL_LEVEL_INVALID`.

**SQL xem dữ liệu CF-02:**
```sql
-- Tài khoản, vai trò, phòng ban, chức danh, trạng thái khóa
SELECT u.email, u.full_name, r.code AS role, u.status, d.name AS department, j.name AS job_title,
       u.failed_login_count, u.locked_until, u.last_login_at
FROM app_user u
JOIN role r ON r.id = u.role_id
LEFT JOIN department d ON d.id = u.department_id
LEFT JOIN job_title j ON j.id = u.job_title_id
ORDER BY u.created_at;
```
```sql
-- Cây phòng ban kèm số nhân sự
WITH RECURSIVE tree AS (
  SELECT id, name, parent_id, 0 AS depth, name::text AS path FROM department WHERE parent_id IS NULL
  UNION ALL
  SELECT d.id, d.name, d.parent_id, t.depth + 1, t.path || ' > ' || d.name
  FROM department d JOIN tree t ON d.parent_id = t.id
)
SELECT repeat('  ', depth) || name AS phong_ban,
       (SELECT count(*) FROM app_user u WHERE u.department_id = tree.id) AS so_nhan_su
FROM tree ORDER BY path;
```
```sql
-- Kỹ năng của từng người
SELECT u.email, s.name AS skill, us.level
FROM user_skill us JOIN skill s ON s.id = us.skill_id JOIN app_user u ON u.id = us.user_id
ORDER BY u.email, s.name;
```

### 4.2. CF-03 — Dự án, thành viên, bảng Kanban, luật chuyển cột

**Trên giao diện:**
1. Manager → **Dự án** → **Tạo dự án**: tên, mô tả, trạng thái, ngày bắt đầu/kết thúc → Lưu. Dự án mới có
   sẵn bảng 5 cột mặc định.
2. Mở dự án → tab **Thành viên**: thêm Member/Manager (danh sách gợi ý không có Admin, CEO), bỏ thành viên.
   Không bỏ được Manager phụ trách.
3. Tab **Cấu hình bảng**: thêm/đổi tên/xóa cột (2–8 cột), kéo đổi thứ tự, chọn nhóm trạng thái
   (Cần làm / Đang làm / Hoàn thành); bảng luật chuyển cột theo vai trò. Member không được có luật chuyển
   vào cột nhóm Hoàn thành.
4. Member → **Dự án**: chỉ thấy dự án mình tham gia, không có nút tạo hoặc sửa.
5. CEO → **Dự án**: thấy mọi dự án, chỉ xem.
6. Admin → không có menu Dự án. Gõ `/projects` → không có quyền.

**Bằng curl:**
1. Phân quyền xem danh sách:
   ```bash
   for r in admin ceo manager member; do
     echo "$r: $(curl -s $API/projects -H "$(H $r)" | jq -c '.total // .error.code')"
   done
   ```
   Mong đợi: `admin: "FORBIDDEN"`, ba vai trò còn lại trả số dự án.
2. Manager tạo dự án (tên có hậu tố để chạy lại không trùng):
   ```bash
   PNAME="Dự án thử $(date +%s)"
   PID=$(jq -n --arg n "$PNAME" '{name:$n, description:"Kiểm thử thủ công", status:"new", startDate:"2026-12-01", endDate:"2026-12-31"}' |
     curl -s -X POST $API/projects -H "$(H manager)" -H 'Content-Type: application/json' -d @- | jq -r .id)
   echo $PID
   ```
3. Ca lỗi:
   ```bash
   jq -n --arg n "$PNAME" '{name:$n, status:"new"}' | curl -s -X POST $API/projects -H "$(H manager)" -H 'Content-Type: application/json' -d @- | jq -c .error
   curl -s -X POST $API/projects -H "$(H manager)" -H 'Content-Type: application/json' -d '{"name":"Ngày sai","status":"new","startDate":"2026-12-31","endDate":"2026-12-01"}' | jq -c .error
   curl -s -X POST $API/projects -H "$(H manager)" -H 'Content-Type: application/json' -d '{"name":"  ","status":"new"}' | jq -c .error
   curl -s -X POST $API/projects -H "$(H manager)" -H 'Content-Type: application/json' -d '{' | jq -c .error
   curl -s -X POST $API/projects -H "$(H admin)" -H 'Content-Type: application/json' -d '{"name":"Admin tạo","status":"new"}' | jq -c .error
   ```
   Mong đợi theo thứ tự: `PROJECT_NAME_TAKEN`, `PROJECT_DATES_INVALID`, `PROJECT_NAME_INVALID`, `INVALID_JSON`, `FORBIDDEN`.
4. Sửa dự án (trạng thái: `new`, `in_progress`, `paused`, `completed`, `cancelled`). Member sửa → 403:
   ```bash
   jq -n --arg n "$PNAME" '{name:$n, description:"Đã sửa", status:"in_progress", startDate:"2026-12-01", endDate:"2027-01-15"}' |
     curl -s -X PUT $API/projects/$PID -H "$(H manager)" -H 'Content-Type: application/json' -d @- | jq '{status, endDate, manager, memberCount}'
   curl -s -X PUT $API/projects/$PID -H "$(H member)" -H 'Content-Type: application/json' -d '{"name":"x","status":"new"}' | jq -c .error
   ```
5. Người ngoài dự án không thấy dự án (trả 404, không phải 403, để không lộ dự án có tồn tại):
   ```bash
   curl -s $API/projects/$PID -H "$(H member)" | jq -c .error
   curl -s $API/projects/$PID -H "$(H ceo)" | jq -r .name
   ```
   Mong đợi: Member → `PROJECT_NOT_FOUND`; CEO xem được.
6. Thành viên:
   ```bash
   curl -s $API/projects/$PID/member-candidates -H "$(H manager)" | jq -c '.items[] | {fullName, role}'
   MEMBER_ID=$(curl -s $API/auth/me -H "$(H member)" | jq -r .id)
   MANAGER_ID=$(curl -s $API/auth/me -H "$(H manager)" | jq -r .id)
   jq -n --arg u "$MEMBER_ID" '{userId:$u}' | curl -s -o /dev/null -w '%{http_code}\n' -X POST $API/projects/$PID/members -H "$(H manager)" -H 'Content-Type: application/json' -d @-
   jq -n --arg u "$ADMIN_ID" '{userId:$u}' | curl -s -X POST $API/projects/$PID/members -H "$(H manager)" -H 'Content-Type: application/json' -d @- | jq -c .error
   curl -s -X DELETE $API/projects/$PID/members/$MANAGER_ID -H "$(H manager)" | jq -c .error
   curl -s $API/projects/$PID/members -H "$(H manager)" | jq -c '.items[] | {fullName, role}'
   ```
   Mong đợi: thêm Member → `204` (thêm lần nữa vẫn 204); thêm Admin → `PROJECT_MEMBER_INVALID`; bỏ Manager phụ
   trách → `PROJECT_CANNOT_REMOVE_MANAGER`. Bây giờ Member đã thấy dự án (`GET /projects/$PID` trả 200).
7. Bảng Kanban mặc định:
   ```bash
   curl -s $API/projects/$PID/board -H "$(H manager)" | tee /tmp/board.json |
     jq -c '.columns[] | {name, statusGroup, position}'
   jq '[.transitions[].role] | group_by(.) | map({(.[0]): length}) | add' /tmp/board.json
   ```
   Mong đợi: 5 cột, mỗi cột có `name`, `statusGroup`, `position`; luật chuyển gồm `manager` và `member`.
8. Cột — thêm, đổi tên, trùng tên, nhóm sai, Member thêm:
   ```bash
   COL=$(curl -s -X POST $API/projects/$PID/columns -H "$(H manager)" -H 'Content-Type: application/json' -d '{"name":"Kiểm thử","statusGroup":"in_progress"}' | jq -r '.columns[] | select(.name=="Kiểm thử") | .id')
   curl -s -X PUT $API/projects/$PID/columns/$COL -H "$(H manager)" -H 'Content-Type: application/json' -d '{"name":"QA","statusGroup":"in_progress"}' | jq -c '[.columns[].name]'
   curl -s -X POST $API/projects/$PID/columns -H "$(H manager)" -H 'Content-Type: application/json' -d '{"name":"QA","statusGroup":"in_progress"}' | jq -c .error
   curl -s -X POST $API/projects/$PID/columns -H "$(H manager)" -H 'Content-Type: application/json' -d '{"name":"Lạ","statusGroup":"blocked"}' | jq -c .error
   curl -s -X POST $API/projects/$PID/columns -H "$(H member)" -H 'Content-Type: application/json' -d '{"name":"M","statusGroup":"todo"}' | jq -c .error
   ```
   Mong đợi: danh sách tên có `QA`; rồi `COLUMN_NAME_TAKEN`, `COLUMN_STATUS_GROUP_INVALID`, `FORBIDDEN`.
9. Đổi thứ tự cột: phải gửi **đủ mọi cột, mỗi cột một lần**:
   ```bash
   IDS=$(curl -s $API/projects/$PID/board -H "$(H manager)" | jq -c '[.columns[].id] | reverse')
   jq -n --argjson ids "$IDS" '{columnIds:$ids}' | curl -s -X PUT $API/projects/$PID/column-order -H "$(H manager)" -H 'Content-Type: application/json' -d @- | jq -c '[.columns[].name]'
   jq -n --argjson ids "$IDS" '{columnIds:$ids[1:]}' | curl -s -X PUT $API/projects/$PID/column-order -H "$(H manager)" -H 'Content-Type: application/json' -d @- | jq -c .error
   ```
   Mong đợi: thứ tự đảo ngược; rồi `COLUMN_ORDER_INVALID`.
10. Xóa cột vừa thêm → trả bảng mới. Bảng còn 2 cột mà xóa tiếp, hoặc thêm cột thứ 9 → `COLUMN_COUNT_INVALID`.
    Cột còn công việc → `COLUMN_HAS_TASKS`.
    ```bash
    curl -s -X DELETE $API/projects/$PID/columns/$COL -H "$(H manager)" | jq '.columns | length'
    ```
11. Luật chuyển cột. `PUT` **thay toàn bộ** tập luật. Member chuyển vào cột nhóm Hoàn thành bị chặn:
    ```bash
    TODO=$(jq -r '[.columns[] | select(.statusGroup=="todo")][0].id' /tmp/board.json)
    DONE=$(jq -r '[.columns[] | select(.statusGroup=="done")][0].id' /tmp/board.json)
    jq -n --arg f "$TODO" --arg t "$DONE" '{transitions:[{fromColumnId:$f, toColumnId:$t, role:"member"}]}' |
      curl -s -X PUT $API/projects/$PID/transitions -H "$(H manager)" -H 'Content-Type: application/json' -d @- | jq -c .error
    jq -n --arg f "$TODO" --arg t "$DONE" '{transitions:[{fromColumnId:$f, toColumnId:$t, role:"manager"}]}' |
      curl -s -X PUT $API/projects/$PID/transitions -H "$(H manager)" -H 'Content-Type: application/json' -d @- | jq '.transitions | length'
    ```
    Mong đợi: `TRANSITION_MEMBER_TO_DONE` "Chỉ Manager được chuyển công việc sang nhóm Hoàn thành"; rồi `1`.

**SQL xem dữ liệu CF-03:**
```sql
-- Dự án, Manager phụ trách, số thành viên
SELECT p.name, p.status, p.start_date, p.end_date, m.full_name AS manager,
       (SELECT count(*) FROM project_member pm WHERE pm.project_id = p.id) AS so_thanh_vien
FROM project p JOIN app_user m ON m.id = p.manager_id
ORDER BY p.created_at;
```
```sql
-- Thành viên từng dự án
SELECT p.name AS du_an, u.full_name, r.code AS role, pm.added_at
FROM project_member pm
JOIN project p ON p.id = pm.project_id
JOIN app_user u ON u.id = pm.user_id
JOIN role r ON r.id = u.role_id
ORDER BY p.name, u.full_name;
```
```sql
-- Cột của từng bảng theo thứ tự
SELECT p.name AS du_an, c.position, c.name AS cot, c.status_group
FROM board_column c JOIN project p ON p.id = c.project_id
ORDER BY p.name, c.position;
```
```sql
-- Luật chuyển cột của dự án mẫu "Cổng thông tin nội bộ"
SELECT f.name AS tu_cot, t.name AS den_cot, w.role
FROM workflow_transition w
JOIN board_column f ON f.id = w.from_column_id
JOIN board_column t ON t.id = w.to_column_id
JOIN project p ON p.id = w.project_id
WHERE p.name = 'Cổng thông tin nội bộ'
ORDER BY w.role, f.position, t.position;
```

### 4.3. CF-06 — Công việc cha, nhận việc, Story Point, nhãn, Công việc của tôi

**Trên giao diện:**
1. Manager → mở dự án mẫu **Cổng thông tin nội bộ** → tab **Công việc**: bảng công việc, lọc theo nhóm
   trạng thái/người nhận/nhãn, tìm kiếm, phân trang.
2. **Tạo công việc**: tiêu đề, mô tả, Story Point (1, 2, 3, 5, 8, 13, 21), người nhận, hạn chót, nhãn. Khi chọn
   Story Point, form hiển thị giờ ước tính (ví dụ 5 SP × 4 giờ = 20h). Công việc mới nằm ở cột đầu tiên của
   nhóm Cần làm (Backlog).
3. Mở chi tiết → **Sửa**: đổi Story Point thì giờ ước tính được tính lại theo hệ số hiện tại.
4. Member → mở công việc chưa giao ở nhóm Cần làm → **Nhận việc** → toast "Đã nhận việc". Member không có
   nút tạo hoặc sửa.
5. Member/Manager → **Công việc của tôi** (`/my-tasks`): bảng các việc được giao, chip WIP (chỉ đếm việc cha ở
   nhóm Đang làm). Admin/CEO mở trang này → không có quyền.
6. Admin → Quản trị → tab **Nhãn**: thêm, đổi tên, xóa (nhãn đang gắn vào công việc → nút Xóa bị khóa kèm lý do).
   Tab **Cấu hình**: số giờ cho 1 Story Point (1–40, mặc định 4).
7. Bảng Kanban, kéo thả, chuyển cột: xem §4.4.

**Bằng curl** (dùng dự án mẫu):
1. Lấy ID dự án mẫu, nhãn, Member:
   ```bash
   SEED_PID=$(curl -s $API/projects -H "$(H manager)" | jq -r '.items[] | select(.name=="Cổng thông tin nội bộ") | .id')
   LABEL=$(curl -s $API/labels -H "$(H manager)" | jq -r '.items[0].id')
   MEMBER_ID=$(curl -s $API/auth/me -H "$(H member)" | jq -r .id)
   echo "$SEED_PID $LABEL $MEMBER_ID"
   ```
2. Danh sách công việc của dự án và các bộ lọc:
   ```bash
   curl -s "$API/projects/$SEED_PID/tasks?pageSize=20" -H "$(H manager)" | jq -c '.items[] | {title, storyPoint, column: .column.name, assignee: .assignee.name}'
   curl -s "$API/projects/$SEED_PID/tasks?statusGroup=todo&unassigned=true" -H "$(H member)" | jq .total
   curl -s "$API/projects/$SEED_PID/tasks?labelId=$LABEL" -H "$(H manager)" | jq .total
   curl -s -G "$API/projects/$SEED_PID/tasks" --data-urlencode 'search=đăng nhập' -H "$(H manager)" | jq '[.items[].title]'
   curl -s "$API/projects/$SEED_PID/tasks?pageSize=500" -H "$(H manager)" | jq .pageSize
   ```
   `pageSize` lớn hơn 100 bị kẹp về `100` (không báo lỗi).
3. Manager tạo công việc 5 Story Point:
   ```bash
   TID=$(jq -n --arg t "Việc thử $(date +%s)" --arg l "$LABEL" '{title:$t, description:"Kiểm thử thủ công", storyPoint:5, dueDate:"2026-12-20", labelIds:[$l]}' |
     curl -s -X POST $API/projects/$SEED_PID/tasks -H "$(H manager)" -H 'Content-Type: application/json' -d @- |
     tee /tmp/task.json | jq -r .id)
   jq '{title, storyPoint, estimatedMinutes, column: .column.name, assignee, labels: [.labels[].name]}' /tmp/task.json
   ```
   Mong đợi: `estimatedMinutes` = 5 × 4 × 60 = `1200`, `column` = `Backlog`, `assignee` = `null`.
4. Ca lỗi khi tạo:
   ```bash
   T="$API/projects/$SEED_PID/tasks"
   curl -s -X POST $T -H "$(H manager)" -H 'Content-Type: application/json' -d '{"title":"SP sai","storyPoint":4}' | jq -c .error
   curl -s -X POST $T -H "$(H manager)" -H 'Content-Type: application/json' -d '{"title":"   "}' | jq -c .error
   curl -s -X POST $T -H "$(H manager)" -H 'Content-Type: application/json' -d '{"title":"Ngày sai","dueDate":"20/12/2026"}' | jq -c .error
   jq -n --arg a "$ADMIN_ID" '{title:"Giao Admin", assigneeId:$a}' | curl -s -X POST $T -H "$(H manager)" -H 'Content-Type: application/json' -d @- | jq -c .error
   curl -s -X POST $T -H "$(H manager)" -H 'Content-Type: application/json' -d '{"title":"Nhãn lạ","labelIds":["00000000-0000-0000-0000-000000000000"]}' | jq -c .error
   curl -s -X POST $T -H "$(H member)" -H 'Content-Type: application/json' -d '{"title":"Member tạo"}' | jq -c .error
   ```
   Mong đợi theo thứ tự: `TASK_STORY_POINT_INVALID`, `TASK_TITLE_INVALID`, `DATE_INVALID`, `TASK_ASSIGNEE_INVALID`, `LABEL_NOT_FOUND`, `FORBIDDEN`.
5. Member nhận việc. Nhận lần hai → 409. Manager nhận → 403:
   ```bash
   curl -s -X POST $API/tasks/$TID/claim -H "$(H member)" | jq '{assignee: .assignee.name, column: .column.name}'
   curl -s -X POST $API/tasks/$TID/claim -H "$(H member)" | jq -c .error
   curl -s -X POST $API/tasks/$TID/claim -H "$(H manager)" | jq -c .error
   ```
   Mong đợi: người nhận là Member, cột **không đổi** (nhận việc ≠ bắt đầu làm); rồi `TASK_ALREADY_ASSIGNED`, `FORBIDDEN`.
6. Manager sửa công việc (PUT thay toàn bộ trường; `labelIds: []` và `dueDate: null` được phép). Member sửa → 403:
   ```bash
   jq -n --arg a "$MEMBER_ID" '{title:"Việc thử (sửa)", description:"", storyPoint:8, assigneeId:$a, dueDate:null, labelIds:[]}' |
     curl -s -X PUT $API/tasks/$TID -H "$(H manager)" -H 'Content-Type: application/json' -d @- | jq '{title, storyPoint, estimatedMinutes, labels}'
   curl -s -X PUT $API/tasks/$TID -H "$(H member)" -H 'Content-Type: application/json' -d '{"title":"x"}' | jq -c .error
   ```
   Mong đợi: `estimatedMinutes` = `1920` (8 × 4 × 60); rồi `FORBIDDEN`.
7. Xem chi tiết theo vai trò:
   ```bash
   for r in admin ceo manager member; do echo "$r: $(curl -s $API/tasks/$TID -H "$(H $r)" | jq -c '.title // .error.code')"; done
   ```
   Mong đợi: `admin: "TASK_NOT_FOUND"`, ba vai trò còn lại thấy tiêu đề.
8. Công việc của tôi và WIP:
   ```bash
   curl -s "$API/me/tasks?statusGroup=todo" -H "$(H member)" | jq -c '{total, titles: [.items[].title]}'
   curl -s $API/me/wip -H "$(H member)" | jq -c .
   curl -s $API/me/tasks -H "$(H ceo)" | jq -c .error
   ```
   Mong đợi: danh sách có "Việc thử (sửa)"; `{"count":…}`; CEO → `FORBIDDEN`.
9. Nhãn (Admin/CEO ghi; Manager/Member chỉ đọc):
   ```bash
   LNAME="Nhãn thử $(date +%s)"
   NEWL=$(jq -n --arg n "$LNAME" '{name:$n}' | curl -s -X POST $API/labels -H "$(H admin)" -H 'Content-Type: application/json' -d @- | jq -r .id)
   jq -n --arg n "$LNAME" '{name:$n}' | curl -s -X POST $API/labels -H "$(H admin)" -H 'Content-Type: application/json' -d @- | jq -c .error
   curl -s -X POST $API/labels -H "$(H manager)" -H 'Content-Type: application/json' -d '{"name":"M"}' | jq -c .error
   curl -s -X DELETE $API/labels/$LABEL -H "$(H admin)" | jq -c .error
   curl -s -o /dev/null -w '%{http_code}\n' -X DELETE $API/labels/$NEWL -H "$(H admin)"
   ```
   Mong đợi: `LABEL_NAME_TAKEN`, `FORBIDDEN`, `LABEL_IN_USE` (nhãn mẫu đang gắn công việc), `204`.
10. Cấu hình giờ/Story Point:
    ```bash
    curl -s $API/settings -H "$(H member)" | jq -c .
    curl -s -X PUT $API/settings -H "$(H admin)" -H 'Content-Type: application/json' -d '{"storyPointHours":50}' | jq -c .error
    curl -s -X PUT $API/settings -H "$(H manager)" -H 'Content-Type: application/json' -d '{"storyPointHours":6}' | jq -c .error
    curl -s -X PUT $API/settings -H "$(H admin)" -H 'Content-Type: application/json' -d '{"storyPointHours":4}' | jq -c .
    ```
    Mong đợi: `{"storyPointHours":4,…}`, `SETTING_INVALID`, `FORBIDDEN`, rồi lưu lại giá trị 4. Đổi hệ số **không**
    sửa giờ ước tính của công việc đã có; chỉ áp dụng khi Story Point được đặt hoặc đổi lần sau.

**SQL xem dữ liệu CF-06:**
```sql
-- Công việc theo dự án, cột, người nhận (view task_view gộp sẵn tên dự án, cột, người nhận)
SELECT project_name, column_name, status_group, title, story_point, estimated_minutes, assignee_name, due_date
FROM task_view
WHERE parent_task_id IS NULL
ORDER BY project_name, column_position, position;
```
```sql
-- Nhãn của từng công việc
SELECT t.title, string_agg(l.name, ', ' ORDER BY l.name) AS nhan
FROM task t JOIN task_label tl ON tl.task_id = t.id JOIN label l ON l.id = tl.label_id
GROUP BY t.id, t.title ORDER BY t.title;
```
```sql
-- WIP của từng người: chỉ đếm việc cha ở nhóm in_progress (CLAUDE.md §5.6)
SELECT assignee_name, count(*) AS wip
FROM task_view
WHERE parent_task_id IS NULL AND status_group = 'in_progress' AND assignee_id IS NOT NULL
GROUP BY assignee_name;
```
```sql
-- Việc Member có thể nhận: chưa giao, nhóm todo, là việc cha
SELECT project_name, title, story_point
FROM task_view
WHERE assignee_id IS NULL AND status_group = 'todo' AND parent_task_id IS NULL;
```
```sql
-- Hệ số giờ/Story Point và đối chiếu giờ ước tính đã chụp
SELECT t.title, t.story_point, t.estimated_minutes, t.story_point * s.int_value * 60 AS theo_he_so_hien_tai
FROM task t JOIN system_setting s ON s.key = 'story_point_hours'
WHERE t.story_point IS NOT NULL;
```
```sql
-- Nhãn và số công việc đang dùng
SELECT l.name, count(tl.task_id) AS so_cong_viec
FROM label l LEFT JOIN task_label tl ON tl.label_id = l.id
GROUP BY l.id, l.name ORDER BY l.name;
```

### 4.4. CF-04, CF-05 — Kanban kéo thả, luật chuyển cột, WIP, realtime

**Trên giao diện** (mở 2 cửa sổ trình duyệt: cửa sổ thường đăng nhập Manager, cửa sổ ẩn danh đăng nhập Member):
1. Manager → dự án **Cổng thông tin nội bộ** → tab **Kanban** (ngay sau tab Thông tin). Bảng có 5 cột: Backlog,
   To Do, In Progress, Chờ nghiệm thu, Done. Đầu cột hiển thị số việc ("3 việc"); trên bảng có chip `WIP x/5`, chỉ báo "Đang cập nhật tức thời"
   và số người đang xem bảng.
2. Kéo một thẻ từ Backlog sang To Do. Trong lúc kéo, cột thả được viền xanh, cột bị chặn mờ đi kèm lý do.
   Thả xong thẻ đổi cột ngay (cập nhật lạc quan), toast "Đã chuyển thẻ sang “To Do”" kèm nút **Hoàn tác**
   (nút chỉ hiện khi đường quay lại hợp lệ theo luật).
3. Ở cửa sổ Member, thẻ đổi cột **không cần tải lại trang**, chậm nhất 2 giây (KPI 1).
4. Member thấy gợi ý "Bạn kéo được thẻ đang giao cho mình." Thẻ của người khác hoặc thẻ chưa giao không kéo được.
   Kéo thẻ của mình về phía **Done** → cột Done bị chặn kèm lý do chỉ Quản lý dự án được chuyển vào nhóm "Hoàn thành".
5. Đổi thứ tự thẻ trong cùng một cột: kéo xuống một bậc thì thẻ nằm sau thẻ đích. Tải lại trang, thứ tự vẫn giữ.
6. **WIP đầy:** Manager phụ trách đặt ô **Giới hạn WIP** trên bảng (1–20, nhập 0 hoặc 21 → báo lỗi ngay dưới ô)
   bằng số việc Member đang làm → **Lưu**. Member kéo thêm một việc vào In Progress → hộp thoại
   "Không thể nhận thêm việc" và thẻ quay về chỗ cũ. Member không thấy ô Giới hạn WIP.
   Đặt lại giới hạn 5 khi thử xong.
7. **Xung đột:** cả hai cửa sổ cùng mở bảng, tắt mạng một cửa sổ (DevTools → Network → Offline), kéo thẻ ở cửa sổ
   còn lại, rồi bật mạng và kéo cùng thẻ đó ngay → dải lỗi báo xung đột kèm "Thẻ đã trở về cột cũ.", bảng tải lại vị trí mới nhất.
8. **Mất kết nối:** dừng API (`Ctrl+C` ở `make dev-local`) → bảng hiện dải "Mất kết nối cập nhật tức thời… đang
   kết nối lại…". Chạy lại API → dải biến mất, bảng tự tải lại.
9. CEO mở bảng: dòng "Bạn chỉ xem bảng này…", không kéo được thẻ nào. Admin không xem được dự án (404).

**Bằng curl** (cần biến, hàm `H` ở §2.2 và `SEED_PID` ở §4.3 bước 1):
1. Lấy ID các cột và tạo hàm tiện ích (`ver` đọc version hiện tại của thẻ; `move` thả thẻ vào đầu cột):
   ```bash
   col() { curl -s $API/projects/$SEED_PID/board -H "$(H manager)" | jq -r --arg n "$1" '.columns[] | select(.name==$n) | .id'; }
   TODO=$(col "To Do"); DOING=$(col "In Progress"); REVIEW=$(col "Chờ nghiệm thu"); DONE=$(col Done)
   ver()  { curl -s $API/tasks/$1 -H "$(H manager)" | jq .version; }
   move() { jq -n --arg c "$3" --argjson v "${4:-$(ver $2)}" '{columnId:$c, position:0, version:$v}' |
            curl -s -X POST $API/tasks/$2/move -H "$(H $1)" -H 'Content-Type: application/json' -d @-; }
   curl -s $API/projects/$SEED_PID/board/tasks -H "$(H member)" | jq -c '[.items[] | {title, column: .column.name, version}] | .[:5]'
   ```
2. Manager tạo việc, Member nhận và kéo sang In Progress:
   ```bash
   K1=$(curl -s -X POST $API/projects/$SEED_PID/tasks -H "$(H manager)" -H 'Content-Type: application/json' \
        -d '{"title":"Kanban thử","storyPoint":3}' | jq -r .id)
   curl -s -X POST $API/tasks/$K1/claim -H "$(H member)" | jq -c '{version}'
   move member $K1 $DOING | jq -c '{column: .column.name, version}'
   ```
   Mong đợi: nhận việc trả `{"version":2}` (mọi thay đổi thẻ đều tăng version); kéo xong `column` = `In Progress`, `version` = `3`.
3. Các ca bị chặn:
   ```bash
   move member  $K1 $DONE   | jq -c .error.code   # Member kéo vào Done
   move member  $K1 $REVIEW 1 | jq -c .error.code # version cũ
   move ceo     $K1 $TODO   | jq -c .error.code   # CEO chỉ xem
   move admin   $K1 $TODO   | jq -c .error.code   # Admin không thấy công việc
   K2=$(curl -s -X POST $API/projects/$SEED_PID/tasks -H "$(H manager)" -H 'Content-Type: application/json' -d '{"title":"Chưa giao"}' | jq -r .id)
   move member  $K2 $TODO   | jq -c .error.code   # Member kéo thẻ không phải của mình
   ```
   Mong đợi theo thứ tự: `TRANSITION_MEMBER_TO_DONE` (422), `TASK_VERSION_CONFLICT` (409), `FORBIDDEN` (403),
   `TASK_NOT_FOUND` (404), `TASK_NOT_ASSIGNEE` (403).
4. Member kéo sang Chờ nghiệm thu, Manager kéo sang Done:
   ```bash
   move member  $K1 $REVIEW | jq -c '{column: .column.name}'
   move manager $K1 $DONE   | jq -c '{column: .column.name, version}'
   ```
5. Giới hạn WIP (chỉ Manager phụ trách đổi, khoảng 1–20):
   ```bash
   curl -s -X PUT $API/projects/$SEED_PID/wip-limit -H "$(H manager)" -H 'Content-Type: application/json' -d '{"wipLimit":0}' | jq -c .error.code
   curl -s -X PUT $API/projects/$SEED_PID/wip-limit -H "$(H member)"  -H 'Content-Type: application/json' -d '{"wipLimit":3}' | jq -c .error.code
   curl -s -X POST $API/tasks/$K2/claim -H "$(H member)" > /dev/null
   W=$(curl -s $API/me/wip -H "$(H member)" | jq '[.count, 1] | max')
   curl -s -X PUT $API/projects/$SEED_PID/wip-limit -H "$(H manager)" -H 'Content-Type: application/json' -d "{\"wipLimit\":$W}" | jq -c '{wipLimit}'
   move member $K2 $DOING | jq -c .error.code
   curl -s -X PUT $API/projects/$SEED_PID/wip-limit -H "$(H manager)" -H 'Content-Type: application/json' -d '{"wipLimit":5}' | jq -c '{wipLimit}'
   ```
   Mong đợi: `WIP_LIMIT_INVALID`, `FORBIDDEN`, giới hạn bằng số việc đang làm, `TASK_WIP_LIMIT` (409), rồi trả về `5`.
   Nếu Member đang có 0 việc đang làm, giới hạn đặt là 1 và lần kéo này **được** (chưa vượt): kéo thêm một việc
   nữa để thấy 409.
6. WebSocket: không token → `401`; Admin (không xem được dự án) → `404`. Kết nối thật kiểm bằng giao diện (bước UI 3)
   hoặc script KPI ở bước 7.
   ```bash
   curl -s -o /dev/null -w '%{http_code}\n' "$API/projects/$SEED_PID/ws"
   curl -s -o /dev/null -w '%{http_code}\n' "$API/projects/$SEED_PID/ws?access_token=$(cat /tmp/tok-admin)"
   ```
7. Đo KPI 1 (2 client WebSocket, ≥ 30 lượt, p95 ≤ 2 giây). Mật khẩu lấy từ output `make seed`, **không** ghi vào file:
   <!-- verify:skip (cần mật khẩu) -->
   ```bash
   KPI_MANAGER_PASSWORD='…' KPI_MEMBER_PASSWORD='…' KPI_RUNS=30 node scripts/kpi/kanban-sync-latency.mjs
   ```
   Mong đợi: dòng cuối `KPI 1 ĐẠT: p95 = … ms ≤ 2000 ms`. Kết quả đã đo: `docs/testing/evidence/kpi1-kanban-sync-latency.md`.

**SQL xem dữ liệu CF-04, CF-05:**
```sql
-- Thứ tự thẻ trên bảng và version (version tăng mỗi lần thẻ thay đổi)
SELECT column_position, column_name, position, title, assignee_name, version
FROM task_view
WHERE project_name = 'Cổng thông tin nội bộ' AND parent_task_id IS NULL
ORDER BY column_position, position;
```
```sql
-- Lịch sử chuyển cột (chỉ ghi khi đổi cột, không ghi khi đổi thứ tự trong cột)
SELECT h.moved_at, t.title, fc.name AS tu_cot, tc.name AS sang_cot, u.full_name AS nguoi_keo
FROM task_move_history h
JOIN task t ON t.id = h.task_id
LEFT JOIN board_column fc ON fc.id = h.from_column_id
LEFT JOIN board_column tc ON tc.id = h.to_column_id
JOIN app_user u ON u.id = h.moved_by
ORDER BY h.moved_at DESC LIMIT 20;
```
```sql
-- Giới hạn WIP từng dự án và số việc đang làm của từng người trong dự án đó
SELECT p.name, p.wip_limit, v.assignee_name, count(v.id) AS dang_lam
FROM project p
LEFT JOIN task_view v ON v.project_id = p.id AND v.parent_task_id IS NULL
  AND v.status_group = 'in_progress' AND v.assignee_id IS NOT NULL
GROUP BY p.name, p.wip_limit, v.assignee_name ORDER BY p.name;
```

---

### 4.5. CF-07 — Việc con, bình luận, thông báo

**Trên giao diện** (cửa sổ thường đăng nhập Manager, cửa sổ ẩn danh đăng nhập Member):
1. Manager tạo một công việc trong **Cổng thông tin nội bộ** và giao cho Member. Ở cửa sổ Member, trong vòng
   30 giây (hoặc tải lại trang) **chuông** trên Topbar có chấm đỏ, nhãn đọc "Thông báo, 1 chưa đọc".
2. Member bấm chuông → bảng thông báo hiện câu "… đã giao cho bạn công việc …". Bấm vào thông báo → được đánh dấu
   đã đọc, số chưa đọc giảm. Nút **Đánh dấu đọc hết** báo toast. `Escape` đóng bảng.
3. Member → **Công việc của tôi** → mở công việc vừa nhận. Hộp chi tiết có khối **Công việc con** (trống: "Chưa có công việc con").
   Bấm **Thêm việc con** → bấm **Lưu** khi chưa nhập tên → lỗi ngay dưới ô. Nhập tên → **Lưu** → toast, việc con hiện
   trong danh sách, **không có Story Point**.
4. Đổi cột việc con qua ô chọn (vd. sang Done) → toast, dòng tiến độ "1/1 việc con đã xong". Member tự chuyển được cả
   sang Done vì việc con không đi qua luật chuyển cột (R04).
5. Bấm nút xóa việc con → hỏi xác nhận ngay trong danh sách → **Xóa** → toast.
6. Mở chi tiết một **việc con** → không có khối Công việc con (chỉ 1 cấp), chỉ có khối Bình luận.
7. Khối **Bình luận**: gửi rỗng → lỗi "Vui lòng nhập nội dung bình luận."; gõ nội dung → bộ đếm ký tự (tối đa 2000) → **Gửi** →
   toast, ô nhập được xóa, bình luận hiện cuối danh sách (cũ nhất ở trên) kèm tên tác giả.
8. Ở cửa sổ Manager, chuông báo "… đã bình luận trong …" kèm trích đoạn. Người viết bình luận không tự nhận thông báo.
9. Member mở một công việc **không giao cho mình** trong dự án: thấy danh sách việc con chỉ ở dạng đọc (không có nút thêm,
   không có ô chọn cột), vẫn bình luận được. CEO mở chi tiết: chỉ xem, không có ô bình luận.

**Bằng curl** (cần `H`, `$API` ở §2.2; `SEED_PID`, `MEMBER_ID` ở §4.3 bước 1; hàm `col` ở §4.4 bước 1):
1. Manager tạo việc cha giao cho Member, Member tạo việc con:
   ```bash
   P1=$(jq -n --arg a "$MEMBER_ID" '{title:"Cha CF-07", storyPoint:5, assigneeId:$a}' |
        curl -s -X POST $API/projects/$SEED_PID/tasks -H "$(H manager)" -H 'Content-Type: application/json' -d @- | jq -r .id)
   S1=$(curl -s -X POST $API/tasks/$P1/subtasks -H "$(H member)" -H 'Content-Type: application/json' \
        -d '{"title":"Con 1","description":""}' | jq -r .id)
   curl -s $API/tasks/$P1/subtasks -H "$(H member)" | jq -c '[.items[] | {title, parentId, storyPoint, column: .column.name}]'
   ```
   Mong đợi: một phần tử, `parentId` = `$P1`, `storyPoint` = `null`.
2. Member chuyển việc con sang Done (không cần version, không qua luật chuyển cột):
   ```bash
   DONE=$(col Done)
   jq -n --arg c "$DONE" '{columnId:$c}' | curl -s -X POST $API/subtasks/$S1/move -H "$(H member)" \
     -H 'Content-Type: application/json' -d @- | jq -c '{column: .column.name}'
   ```
   Mong đợi: `{"column":"Done"}`.
3. Các ca bị chặn:
   ```bash
   # Con của con → 422 SUBTASK_PARENT_NOT_ROOT
   curl -s -X POST $API/tasks/$S1/subtasks -H "$(H member)" -H 'Content-Type: application/json' -d '{"title":"Cháu","description":""}' | jq -c .error.code
   # Tên rỗng → 400 VALIDATION_FAILED
   curl -s -X POST $API/tasks/$P1/subtasks -H "$(H member)" -H 'Content-Type: application/json' -d '{"title":"","description":""}' | jq -c .error.code
   # Việc không giao cho Member → 403 SUBTASK_NOT_ALLOWED
   K3=$(curl -s -X POST $API/projects/$SEED_PID/tasks -H "$(H manager)" -H 'Content-Type: application/json' -d '{"title":"Không phải của Member"}' | jq -r .id)
   curl -s -X POST $API/tasks/$K3/subtasks -H "$(H member)" -H 'Content-Type: application/json' -d '{"title":"x","description":""}' | jq -c .error.code
   # CEO chỉ xem → 403; Admin không thấy dự án → 404
   curl -s -X POST $API/tasks/$P1/subtasks -H "$(H ceo)"   -H 'Content-Type: application/json' -d '{"title":"x","description":""}' | jq -c .error.code
   curl -s -X POST $API/tasks/$P1/subtasks -H "$(H admin)" -H 'Content-Type: application/json' -d '{"title":"x","description":""}' | jq -c .error.code
   ```
4. Bình luận và thông báo:
   ```bash
   curl -s -X POST $API/tasks/$P1/comments -H "$(H member)" -H 'Content-Type: application/json' -d '{"body":"Xong phần API"}' | jq -c '{author: .author.name, body}'
   curl -s -X POST $API/tasks/$P1/comments -H "$(H member)" -H 'Content-Type: application/json' -d '{"body":"   "}' | jq -c .error.code   # 422 COMMENT_INVALID
   python3 -c 'print("{\"body\":\"" + "á"*2001 + "\"}")' |
     curl -s -X POST $API/tasks/$P1/comments -H "$(H member)" -H 'Content-Type: application/json' -d @- | jq -c .error.code  # 422 COMMENT_INVALID
   curl -s $API/tasks/$P1/comments -H "$(H manager)" | jq -c '[.items[] | .body]'
   curl -s $API/me/notifications/unread-count -H "$(H manager)" | jq .count            # ≥ 1 (bình luận)
   curl -s "$API/me/notifications?unread=true" -H "$(H member)" | jq -c '[.items[] | {type, task: .task.name, read}]'  # có task_assigned
   ```
   Mong đợi: 2001 ký tự có dấu bị chặn (đếm theo ký tự, không theo byte; 2000 ký tự thì được).
5. Đánh dấu đã đọc:
   ```bash
   N1=$(curl -s "$API/me/notifications?unread=true" -H "$(H member)" | jq -r '.items[0].id')
   curl -s -o /dev/null -w '%{http_code}\n' -X POST $API/me/notifications/$N1/read -H "$(H member)"     # 204
   curl -s -X POST $API/me/notifications/$N1/read -H "$(H manager)" | jq -c .error.code                 # 404: thông báo của người khác
   curl -s -o /dev/null -w '%{http_code}\n' -X POST $API/me/notifications/read-all -H "$(H member)"    # 204
   curl -s $API/me/notifications/unread-count -H "$(H member)" | jq .count                              # 0
   ```
6. Xóa việc con:
   ```bash
   curl -s -o /dev/null -w '%{http_code}\n' -X DELETE $API/subtasks/$S1 -H "$(H member)"   # 204
   ```

**Xem trong CSDL:**
```sql
-- Việc con và việc cha; con không có Story Point
SELECT c.title AS viec_con, p.title AS viec_cha, c.story_point, bc.name AS cot
FROM task c JOIN task p ON p.id = c.parent_task_id JOIN board_column bc ON bc.id = c.column_id
ORDER BY c.created_at DESC LIMIT 10;
```
```sql
-- Ràng buộc ở CSDL: con có Story Point và con của con đều bị chặn (chạy trong giao dịch rồi ROLLBACK)
BEGIN;
UPDATE task SET story_point = 3 WHERE parent_task_id IS NOT NULL;     -- lỗi CHECK (23514)
ROLLBACK;
```
```sql
-- Bình luận và thông báo mới nhất
SELECT t.title, u.full_name, left(c.body, 40) AS noi_dung, c.created_at
FROM task_comment c JOIN task t ON t.id = c.task_id JOIN app_user u ON u.id = c.author_id
ORDER BY c.created_at DESC LIMIT 10;

SELECT u.full_name AS nguoi_nhan, n.type, left(n.message, 40) AS trich, n.read_at, n.created_at
FROM notification n JOIN app_user u ON u.id = n.user_id
ORDER BY n.created_at DESC LIMIT 10;
```

---

## 5. Xem CSDL bằng psql và pgAdmin

### 5.1. psql
1. Kết nối (cùng URL với `DATABASE_URL`):
   <!-- verify:skip (phiên tương tác) -->
   ```bash
   psql "postgres://worktrack:worktrack@localhost:5432/worktrack?sslmode=disable"
   ```
   Nếu máy không tìm thấy `psql`: `/opt/homebrew/opt/postgresql@16/bin/psql`.
2. Lệnh hay dùng trong psql:

   | Lệnh | Ý nghĩa |
   |---|---|
   | `\dt` | Liệt kê bảng (schema `public`) |
   | `\dv` | Liệt kê view (`task_view`) |
   | `\d app_user` | Cột, kiểu, index, khóa ngoại, CHECK của bảng `app_user` |
   | `\d task` | Như trên cho `task`; xem các CHECK `ck_task_story_point_parent_only`, `ck_task_story_point_range` |
   | `\x` | Bật/tắt hiển thị dọc (dễ đọc dòng dài) |
   | `\q` | Thoát |

   Có thể chạy một lệnh mà không vào psql:
   ```bash
   psql "postgres://worktrack:worktrack@localhost:5432/worktrack?sslmode=disable" -c '\dt'
   psql "postgres://worktrack:worktrack@localhost:5432/worktrack?sslmode=disable" -c '\d task'
   ```
   > Trên máy dev có thể thấy thêm các schema `it_<chuỗi hex>`. Đó là schema tạm do test tích hợp tạo,
   > không ảnh hưởng dữ liệu `public`.
3. Phiên bản migration đã áp dụng:
   ```sql
   SELECT version_id, is_applied, tstamp FROM goose_db_version ORDER BY id;
   ```

### 5.2. Bảng nào chứa gì

| Bảng / view | Nội dung | Chức năng |
|---|---|---|
| `role` | 4 vai trò (`id`, `code`: admin, manager, member, ceo) | CF-01 |
| `app_user` | Tài khoản: email, họ tên, vai trò, trạng thái (`pending`/`active`), hash mật khẩu, số lần sai, `locked_until`, token kích hoạt (dạng hash), phòng ban, chức danh | CF-01, CF-02 |
| `refresh_token` | Phiên làm mới (hash token, `family_id`, hạn, `revoked_at`) | CF-01 |
| `department` | Phòng ban dạng cây (`parent_id`) | CF-02 |
| `job_title` | Danh mục chức danh | CF-02 |
| `skill` | Danh mục kỹ năng | CF-02 |
| `user_skill` | Kỹ năng của từng người + cấp độ | CF-02 |
| `project` | Dự án, trạng thái, ngày, Manager phụ trách | CF-03 |
| `project_member` | Thành viên dự án | CF-03 |
| `board_column` | Cột bảng Kanban của từng dự án (`position`, `status_group`) | CF-03 |
| `workflow_transition` | Luật chuyển cột theo vai trò | CF-03 |
| `task` | Công việc (cha/con qua `parent_task_id`), Story Point, giờ ước tính, cột, người nhận, hạn | CF-06 |
| `task_view` (view) | `task` kèm tên dự án, tên/nhóm cột, tên người nhận | CF-06 |
| `label`, `task_label` | Danh mục nhãn và nhãn gắn vào công việc | CF-06 |
| `system_setting` | Tham số hệ thống: `story_point_hours` (`int_value`) | CF-06 |
| `goose_db_version` | Lịch sử migration (goose quản lý — không sửa tay) | — |

Một vài câu SQL tổng hợp:
```sql
SELECT r.code AS role, u.status, count(*) FROM app_user u JOIN role r ON r.id = u.role_id GROUP BY 1, 2 ORDER BY 1, 2;
```
```sql
-- Phiên refresh còn hiệu lực của từng người
SELECT u.email, count(*) AS phien_con_han
FROM refresh_token rt JOIN app_user u ON u.id = rt.user_id
WHERE rt.revoked_at IS NULL AND rt.expires_at > now()
GROUP BY u.email ORDER BY u.email;
```

### 5.3. pgAdmin
1. Cài pgAdmin 4 (ví dụ `brew install --cask pgadmin4`) rồi mở.
2. Chuột phải **Servers** → **Register** → **Server…**
3. Tab **General** → Name: `WorkTrack dev`.
4. Tab **Connection**:
   - Host name/address: `localhost`
   - Port: `5432`
   - Maintenance database: `worktrack`
   - Username: `worktrack`
   - Password: `worktrack` (tick **Save password** nếu muốn)
5. **Save**. Mở **WorkTrack dev → Databases → worktrack → Schemas → public → Tables** để xem bảng; chuột
   phải một bảng → **View/Edit Data → All Rows**.
6. Chạy SQL ở mục §4: chọn CSDL `worktrack` → **Tools → Query Tool** → dán câu SQL → F5.
7. Chỉ đọc trong pgAdmin. Sửa dữ liệu bằng tay dễ làm lệch quy tắc nghiệp vụ; thay vào đó hãy dùng app,
   API hoặc §7.

---

## 6. Thử API bằng curl

Tóm tắt các bước (chi tiết và ví dụ ở §2.2 và §4):
1. Đăng nhập lấy access token:
   ```bash
   curl -s -c /tmp/wt-manager.jar -X POST http://localhost:8080/api/v1/auth/login \
     -H 'Content-Type: application/json' \
     -d '{"email":"manager@worktrack.local","password":"<MK_MANAGER>"}' | jq -r .accessToken > /tmp/tok-manager
   ```
   - `-c` lưu cookie `wt_refresh` để gọi `/auth/refresh`, `/auth/logout`.
   - Cookie có cờ `Secure`, nhưng curl vẫn gửi nó tới `http://localhost` (đã thử với curl 8.7).
2. Gọi endpoint có header `Authorization: Bearer <token>`:
   ```bash
   curl -s http://localhost:8080/api/v1/projects -H "Authorization: Bearer $(cat /tmp/tok-manager)" | jq '.items[] | {id, name, status}'
   ```
3. Token hết hạn sau 15 phút → mọi request trả 401 `UNAUTHENTICATED`. Làm mới:
   ```bash
   curl -s -b /tmp/wt-manager.jar -c /tmp/wt-manager.jar -X POST http://localhost:8080/api/v1/auth/refresh | jq -r .accessToken > /tmp/tok-manager
   ```
4. Mẹo:
   - Dữ liệu có tiếng Việt trên URL: `curl -G … --data-urlencode 'search=đăng nhập'`.
   - Tạo JSON có biến shell: `jq -n --arg x "$BIEN" '{name:$x}' | curl … -d @-`, để khỏi lo dấu nháy.
   - Xem cả mã HTTP: thêm `-w '\n%{http_code}\n'`, hoặc `-i` để xem header.
   - Danh sách có phân trang trả `{items, page, pageSize, total}`. Tham số `page`, `pageSize` (tối đa 100).

Danh sách endpoint hiện có (`backend/internal/transport/httpapi/router.go`, tiền tố `/api/v1`):

| Nhóm | Endpoint |
|---|---|
| Xác thực | `POST /auth/login`, `POST /auth/refresh`, `POST /auth/logout`, `POST /auth/activation/verify`, `POST /auth/activation`, `GET /auth/me` |
| Tài khoản | `GET/POST /users`, `GET/PUT /users/{id}`, `PUT /users/{id}/skills`, `POST /users/{id}/activation-email` |
| Tổ chức | `GET/POST /departments`, `PUT/DELETE /departments/{id}`; tương tự cho `/job-titles`, `/skills` |
| Dự án | `GET/POST /projects`, `GET/PUT /projects/{id}`, `GET/POST /projects/{id}/members`, `DELETE /projects/{id}/members/{userId}`, `GET /projects/{id}/member-candidates` |
| Bảng | `GET /projects/{id}/board`, `POST /projects/{id}/columns`, `PUT/DELETE /projects/{id}/columns/{columnId}`, `PUT /projects/{id}/column-order`, `PUT /projects/{id}/transitions` |
| Công việc | `GET/POST /projects/{id}/tasks`, `GET/PUT /tasks/{id}`, `POST /tasks/{id}/claim`, `GET /me/tasks`, `GET /me/wip` |
| Nhãn, cấu hình | `GET/POST /labels`, `PUT/DELETE /labels/{id}`, `GET/PUT /settings` |
| Sức khỏe (không tiền tố) | `GET /healthz`, `GET /readyz` |

---

## 7. Đặt lại dữ liệu dev an toàn

Chọn cách **nhẹ nhất** đủ dùng. Mọi cách dưới đây chỉ dành cho CSDL dev `worktrack` trên máy bạn.

### 7.1. Đặt lại mật khẩu và mở khóa tài khoản mẫu
```bash
make seed
```
Không xóa gì. Chỉ đặt lại 4 mật khẩu mẫu, mở khóa 4 tài khoản mẫu, và tạo lại phòng ban, dự án, công việc
mẫu nếu thiếu.

### 7.2. Mở khóa một tài khoản bất kỳ
```sql
UPDATE app_user SET failed_login_count = 0, locked_until = NULL WHERE email = 'member@worktrack.local';
```

### 7.3. Xóa dữ liệu thử, giữ dữ liệu mẫu
Xóa những gì các bước trong tài liệu này tạo ra (tên có tiền tố "Dự án thử", "Việc thử", "Nhãn thử",
"Khối thử", email `nv.*`).

1. Chạy thử trong transaction và **ROLLBACK** để xem số dòng bị xóa trước:
   ```sql
   BEGIN;
   -- Xóa dự án → tự xóa (cascade) thành viên, cột, luật chuyển, công việc, nhãn gắn của dự án đó
   DELETE FROM project WHERE name LIKE 'Dự án thử %' COLLATE "C";
   -- Công việc thử tạo trong dự án mẫu (task_label xóa theo cascade)
   DELETE FROM task WHERE title LIKE 'Việc thử%' COLLATE "C";
   -- Nhãn thử còn sót (nhãn đang gắn công việc không xóa được — khóa ngoại RESTRICT)
   DELETE FROM label WHERE name LIKE 'Nhãn thử %' COLLATE "C";
   -- Phòng ban thử: xóa con trước, cha sau
   DELETE FROM department WHERE name LIKE 'Nhóm con %' COLLATE "C";
   DELETE FROM department WHERE name LIKE 'Khối thử %' COLLATE "C";
   -- Tài khoản thử (refresh_token, user_skill xóa theo cascade).
   -- Sẽ lỗi nếu tài khoản còn là người nhận/người tạo công việc hoặc Manager của dự án: xử lý các dòng đó trước.
   DELETE FROM app_user WHERE email LIKE 'nv.%@worktrack.local';
   SELECT (SELECT count(*) FROM project) AS projects, (SELECT count(*) FROM task) AS tasks, (SELECT count(*) FROM app_user) AS users;
   ROLLBACK;
   ```
2. Thấy đúng như mong muốn thì chạy lại, thay `ROLLBACK;` bằng `COMMIT;`.
   - Vì sao có `COLLATE "C"`: trên macOS, `LIKE 'tiền tố %'` với chữ có dấu (vd. `'Khối thử %'`) dưới collation
     `en_US.UTF-8` có thể làm tiến trình PostgreSQL bị hủy (`Assertion failed … collate.c`, mất kết nối). Lỗi
     nằm ở hàm so sánh chuỗi của libc macOS. `COLLATE "C"` so theo byte nên tránh
     được, và kết quả lọc của LIKE không đổi.
3. Không xóa trực tiếp `board_column`, `workflow_transition`, `role`, `system_setting`, `goose_db_version`.

Quan hệ xóa giữa các bảng (`\d <bảng>` để xem):
- **Cascade** (xóa cha thì con tự xóa):
  - `project` → `project_member`, `board_column`, `task`;
  - `board_column` → `workflow_transition`;
  - `task` → `task_label` và việc con;
  - `app_user` → `refresh_token`, `user_skill`.
- **Restrict** (bị chặn nếu còn tham chiếu):
  - `task` → `app_user` (người nhận, người tạo) và `task` → `board_column`;
  - `project` → `app_user` (Manager);
  - `task_label` → `label`;
  - tham chiếu tới `department`, `job_title`, `skill`.

### 7.4. Làm lại toàn bộ từ đầu (xóa hết dữ liệu)
> ⚠️ **Xóa toàn bộ dữ liệu** trong CSDL dev. Chỉ làm khi thật sự cần. Bước này **không chạy** khi kiểm
> chứng tài liệu (phiên tự động không được xóa dữ liệu). `goose reset`, `goose up`, `make seed` là các lệnh
> có sẵn, đã kiểm bằng `goose --help`.

1. Dừng `make dev-local` (Ctrl+C).
2. Gỡ mọi migration (xóa mọi bảng do migration tạo):
   <!-- verify:skip (xóa dữ liệu) -->
   ```bash
   cd backend && goose -dir db/migrations postgres "postgres://worktrack:worktrack@localhost:5432/worktrack?sslmode=disable" reset; cd ..
   ```
   (`make migrate-down` chỉ gỡ **một** migration mỗi lần.)
3. Tạo lại bảng và dữ liệu mẫu:
   <!-- verify:skip (đi kèm bước 2) -->
   ```bash
   make migrate-up
   make seed
   ```
4. Ghi lại 4 mật khẩu mới, chạy `make dev-local`.

Không dùng `DROP DATABASE`, không xóa thư mục dữ liệu của PostgreSQL. Không chạy các lệnh trên với
`DATABASE_URL` khác máy dev.

---

## 8. Checklist kiểm thử

Đánh dấu `[x]` khi đã thử và đúng như mong đợi. Mã UC/YCCN chi tiết ở `docs/requirements/` và
`docs/testing/traceability-matrix.md`.

### Scaffold / môi trường
- [ ] PostgreSQL và Mailpit `started`; `pg_isready` OK (§1.2)
- [ ] `.env` có `DATABASE_URL`, `JWT_SECRET` ≥ 32 byte (§1.3)
- [ ] `make migrate-up` không còn migration Pending (§1.4)
- [ ] `make seed` in 4 tài khoản (§1.4)
- [ ] `/healthz` → ok, `/readyz` → ready, web 5173 → 200 (§1.5)

### CF-01 — Đăng nhập, phiên, kích hoạt
- [ ] Đăng nhập được 4 vai trò, menu đúng vai trò (§2.1)
- [ ] Sai mật khẩu → câu báo chung, không lộ email (§2.1)
- [ ] `/auth/me` không token → 401 (§2.2 bước 4)
- [ ] F5 vẫn giữ phiên; refresh cấp token mới (§2.2 bước 5)
- [ ] Đăng xuất → 204, refresh sau đó → 401 (§2.2 bước 6)
- [ ] Sai 5 lần → 423 `AUTH_ACCOUNT_LOCKED`; `locked_until` có giá trị; mở khóa được (§2.3)
- [ ] Đăng nhập dồn dập → 429 `RATE_LIMITED` (§2.3 bước 4)
- [ ] Admin tạo tài khoản → email trong Mailpit (§3)
- [ ] Kích hoạt: mật khẩu yếu bị chặn, kích hoạt thành công, link dùng lại bị chặn (§3.2)
- [ ] Tài khoản chờ kích hoạt không đăng nhập được (§3.1 bước 6)

### CF-02 — Cơ cấu tổ chức và tài khoản
- [ ] Manager/Member bị 403 ở phòng ban, tài khoản (§4.1)
- [ ] Phòng ban: tạo cây, chặn vòng lặp, chặn trùng tên cùng cha, chặn xóa khi còn con hoặc còn nhân sự (§4.1 bước 2–4)
- [ ] Chức danh, kỹ năng: chặn trùng tên (không phân biệt hoa thường), chặn xóa khi đang dùng (§4.1 bước 5)
- [ ] Tài khoản: lọc, tìm kiếm, phân trang (§4.1 bước 6)
- [ ] Tạo trùng email, email sai, Admin tạo CEO → bị chặn (§4.1 bước 7)
- [ ] Admin không tự đổi vai trò của mình (§4.1 bước 8)
- [ ] Gán kỹ năng + cấp độ; cấp độ sai bị chặn (§4.1 bước 9)

### CF-03 — Dự án và bảng
- [ ] Admin không xem, không tạo dự án; CEO chỉ xem (§4.2 bước 1, 3, 5)
- [ ] Manager tạo/sửa dự án; trùng tên, ngày sai, tên rỗng, JSON hỏng bị chặn (§4.2 bước 2–4)
- [ ] Member không sửa được dự án; người ngoài dự án nhận 404 (§4.2 bước 4–5)
- [ ] Thêm/bỏ thành viên; không thêm Admin; không bỏ Manager phụ trách (§4.2 bước 6)
- [ ] Dự án mới có 5 cột mặc định và luật chuyển (§4.2 bước 7)
- [ ] Thêm/đổi tên/xóa cột; trùng tên, nhóm sai, quá 2–8 cột bị chặn; Member không sửa bảng (§4.2 bước 8, 10)
- [ ] Đổi thứ tự cột; thiếu cột bị chặn (§4.2 bước 9)
- [ ] Luật chuyển: Member không được vào nhóm Hoàn thành (§4.2 bước 11)

### CF-06 — Công việc cha
- [ ] Danh sách công việc: lọc nhóm, chưa giao, nhãn, tìm kiếm, phân trang (§4.3 bước 2)
- [ ] Manager tạo việc: vào cột Backlog, giờ ước tính = SP × hệ số × 60 phút (§4.3 bước 3)
- [ ] SP ngoài thang Fibonacci, tiêu đề rỗng, ngày sai, người nhận ngoài dự án, nhãn lạ, Member tạo → bị chặn (§4.3 bước 4)
- [ ] Member nhận việc; nhận lần hai → 409; Manager nhận → 403; cột không đổi (§4.3 bước 5)
- [ ] Manager sửa việc, SP mới → giờ tính lại; Member sửa → 403 (§4.3 bước 6)
- [ ] Admin không thấy công việc (404); CEO, Manager, Member thấy (§4.3 bước 7)
- [ ] Công việc của tôi, WIP; CEO/Admin bị 403 (§4.3 bước 8)
- [ ] Nhãn: chỉ Admin/CEO ghi; trùng tên, đang dùng bị chặn (§4.3 bước 9)
- [ ] Cấu hình giờ/SP: 1–40; Manager không sửa được (§4.3 bước 10)

### CF-04, CF-05 — Kanban và WIP
- [ ] Bảng 5 cột, số việc mỗi cột, chip WIP, chỉ báo cập nhật tức thời (§4.4 UI bước 1)
- [ ] Kéo thả: vùng thả hợp lệ/bị chặn có lý do, toast + Hoàn tác (§4.4 UI bước 2)
- [ ] Cửa sổ thứ hai thấy thẻ đổi cột ≤ 2 giây, không tải lại trang (§4.4 UI bước 3)
- [ ] Member chỉ kéo thẻ của mình; vào Done → 422 `TRANSITION_MEMBER_TO_DONE` (§4.4 curl bước 3)
- [ ] Version cũ → 409 `TASK_VERSION_CONFLICT`; CEO kéo → 403; Admin → 404 (§4.4 curl bước 3)
- [ ] Đổi thứ tự trong cột được giữ sau khi tải lại (§4.4 UI bước 5)
- [ ] Giới hạn WIP 1–20, chỉ Manager phụ trách đổi; vượt WIP → 409 `TASK_WIP_LIMIT` + hộp thoại (§4.4 curl bước 5, UI bước 6)
- [ ] Mất kết nối → dải cảnh báo, tự kết nối lại (§4.4 UI bước 8)
- [ ] WebSocket không token → 401, người ngoài dự án → 404 (§4.4 curl bước 6)
- [ ] KPI 1: p95 ≤ 2 giây trên ≥ 30 lượt (§4.4 curl bước 7)
- [ ] `task_move_history` có dòng cho mỗi lần đổi cột (SQL §4.4)

### CF-07 — Việc con, bình luận, thông báo
- [ ] Được giao việc → chuông có chấm đỏ, câu thông báo đúng loại; bấm → đã đọc; đọc hết → toast (§4.5 UI bước 1–2)
- [ ] Người nhận việc cha thêm việc con, không có Story Point; tên rỗng bị chặn (§4.5 UI bước 3, curl bước 1)
- [ ] Member tự chuyển việc con kể cả sang Done; tiến độ "x/y việc con đã xong" (§4.5 UI bước 4, curl bước 2)
- [ ] Con của con → 422 `SUBTASK_PARENT_NOT_ROOT`; việc không phải của mình → 403 `SUBTASK_NOT_ALLOWED`; CEO → 403; Admin → 404 (§4.5 curl bước 3)
- [ ] Xóa việc con có bước xác nhận (§4.5 UI bước 5)
- [ ] Bình luận: rỗng/khoảng trắng hoặc > 2000 ký tự → 422 `COMMENT_INVALID`; cũ nhất ở trên (§4.5 curl bước 4)
- [ ] Có bình luận → người nhận việc và người tạo việc được báo, người viết không tự nhận (§4.5 UI bước 8)
- [ ] Đánh dấu đọc thông báo của người khác → 404 `NOTIFICATION_NOT_FOUND` (§4.5 curl bước 5)
- [ ] CSDL chặn Story Point ở việc con (SQL §4.5)

### Dọn dẹp
- [ ] Chạy SQL §7.3 với `ROLLBACK` trước, rồi `COMMIT`
- [ ] `make seed` để có mật khẩu mới nếu cần (§7.1)
