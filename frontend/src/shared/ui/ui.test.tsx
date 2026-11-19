import { act, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useState } from 'react'
import { Avatar, Badge, Button, Dialog, EmptyState, ErrorState, Input, ToastProvider } from '.'
import { useToast } from './toast/useToast'

describe('Button', () => {
  it('đang xử lý → vô hiệu hóa và báo aria-busy', () => {
    render(<Button loading>Lưu</Button>)
    const btn = screen.getByRole('button', { name: 'Lưu' })
    expect(btn).toBeDisabled()
    expect(btn).toHaveAttribute('aria-busy', 'true')
  })

  it('mặc định type="button" để không vô tình submit form', () => {
    render(<Button>Hủy</Button>)
    expect(screen.getByRole('button', { name: 'Hủy' })).toHaveAttribute('type', 'button')
  })
})

describe('Input', () => {
  it('gắn nhãn và thông báo lỗi vào ô nhập', () => {
    render(<Input label="Email" error="Email chưa đúng định dạng." />)
    const input = screen.getByLabelText('Email')
    expect(input).toHaveAttribute('aria-invalid', 'true')
    expect(input).toHaveAccessibleDescription('Email chưa đúng định dạng.')
  })

  it('không lỗi → hiện gợi ý, aria-invalid false', () => {
    render(<Input label="Mật khẩu" hint="Tối thiểu 8 ký tự." />)
    const input = screen.getByLabelText('Mật khẩu')
    expect(input).toHaveAttribute('aria-invalid', 'false')
    expect(input).toHaveAccessibleDescription('Tối thiểu 8 ký tự.')
  })
})

describe('Badge, Avatar', () => {
  it('Badge hiện chữ (không chỉ dựa vào màu)', () => {
    render(<Badge tone="ok">Đang hoạt động</Badge>)
    expect(screen.getByText('Đang hoạt động')).toBeInTheDocument()
  })

  it('Avatar lấy 2 chữ cái cuối của họ tên tiếng Việt', () => {
    render(<Avatar name="Nguyễn Văn An" />)
    expect(screen.getByRole('img', { name: 'Nguyễn Văn An' })).toHaveTextContent('VA')
  })
})

describe('Dialog', () => {
  function Harness() {
    const [open, setOpen] = useState(true)
    return (
      <Dialog
        open={open}
        onClose={() => {
          setOpen(false)
        }}
        title="Đăng xuất?"
        description="Bạn sẽ cần đăng nhập lại."
      >
        <Button>Hủy</Button>
      </Dialog>
    )
  }

  it('có role dialog, tiêu đề, mô tả và đóng bằng Escape', async () => {
    const user = userEvent.setup()
    render(<Harness />)
    const dialog = screen.getByRole('dialog', { name: 'Đăng xuất?' })
    expect(dialog).toHaveAttribute('aria-modal', 'true')
    expect(dialog).toHaveAccessibleDescription('Bạn sẽ cần đăng nhập lại.')
    await user.keyboard('{Escape}')
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })
})

describe('EmptyState, ErrorState', () => {
  it('EmptyState hiện tiêu đề và hành động', () => {
    render(<EmptyState title="Chưa có dự án" description="Tạo dự án đầu tiên." />)
    expect(screen.getByRole('heading', { name: 'Chưa có dự án' })).toBeInTheDocument()
  })

  it('ErrorState hiện mã lỗi, câu dữ liệu cũ và gọi Thử lại', async () => {
    const user = userEvent.setup()
    const onRetry = vi.fn()
    render(<ErrorState title="Không tải được" errorRef="INTERNAL" onRetry={onRetry} />)
    expect(screen.getByRole('alert')).toHaveTextContent('INTERNAL')
    expect(screen.getByText(/có thể đã cũ/)).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Thử lại' }))
    expect(onRetry).toHaveBeenCalledTimes(1)
  })
})

describe('Toast', () => {
  function Trigger() {
    const toast = useToast()
    return (
      <Button
        onClick={() => {
          toast.show('Đã lưu')
        }}
      >
        Lưu
      </Button>
    )
  }

  it('hiện thông báo một dòng và tự ẩn sau 3 giây', () => {
    vi.useFakeTimers()
    render(
      <ToastProvider>
        <Trigger />
      </ToastProvider>,
    )
    act(() => {
      screen.getByRole('button', { name: 'Lưu' }).click()
    })
    expect(screen.getByRole('status')).toHaveTextContent('Đã lưu')
    act(() => {
      vi.advanceTimersByTime(3000)
    })
    expect(screen.queryByText('Đã lưu')).not.toBeInTheDocument()
  })
})
