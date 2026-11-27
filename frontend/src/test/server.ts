import { setupServer } from 'msw/node'

/** Máy chủ MSW dùng chung cho mọi test; mỗi test tự khai báo handler bằng server.use(). */
export const server = setupServer()

export const API = 'http://localhost:8080'
