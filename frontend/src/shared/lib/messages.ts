// Mọi chữ hiển thị cho người dùng đặt ở đây (tiếng Việt).
export const messages = {
  appName: 'WorkTrack',
  health: {
    title: 'Trạng thái hệ thống',
    checking: 'Đang kiểm tra kết nối máy chủ…',
    ok: 'Máy chủ đang hoạt động',
    error: 'Không kết nối được máy chủ',
    retry: 'Thử lại',
  },
  notFound: 'Không tìm thấy trang',
  errors: {
    network: 'Không kết nối được máy chủ',
    unknown: 'Đã có lỗi xảy ra, vui lòng thử lại sau',
  },
} as const
