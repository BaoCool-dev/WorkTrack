import { http as mock, HttpResponse } from 'msw'
import { setupServer } from 'msw/node'

export const API = 'http://localhost:8080'

/**
 * Máy chủ MSW dùng chung cho mọi test; mỗi test tự khai báo handler bằng server.use().
 * Handler khởi tạo ở đây là mặc định: resetHandlers() giữ lại, server.use() trong test đè lên.
 */
export const server = setupServer(
  // Chuông thông báo ở Topbar (CF-07) gọi trên mọi trang đã đăng nhập.
  mock.get(`${API}/api/v1/me/notifications/unread-count`, () => HttpResponse.json({ count: 0 })),
)
