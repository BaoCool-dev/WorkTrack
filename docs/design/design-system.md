# docs/design/design-system.md — Hệ thống thiết kế WorkTrack

> Nguồn: bản thiết kế **"WorkTrack – Thiết kế giao diện"** (Claude Design, cập nhật 18/11/2026), 13 artboard.
> Bản HTML xuất ra đặt ở `docs/design/artboards/` — khi làm một màn hình, **mở đúng file artboard tương ứng** (bảng §9) và bám theo.
> File này là luật; artboard là ví dụ. Hai bên lệch nhau → theo file này và ghi lại ở §10.
> Không tự đổi màu, font, cỡ chữ, bo góc. Thấy cần đổi → hỏi người dùng.

## 1. Token (Tailwind v4 — `frontend/src/app/styles/theme.css`)

```css
@import "tailwindcss";

@theme {
  --font-sans: "Be Vietnam Pro", ui-sans-serif, system-ui, sans-serif;

  /* Cỡ chữ: CHỈ 3 cỡ */
  --text-sm: 12px;   /* phụ chú, nhãn, tiêu đề cột, badge */
  --text-base: 14px; /* nội dung, ô bảng, nút, tên công việc */
  --text-lg: 20px;   /* tiêu đề trang, số liệu lớn */

  /* Màu nhấn */
  --color-accent: #1D4ED8;
  --color-accent-hover: #1E40AF;        /* cũng là màu chữ của mục đang chọn */
  --color-accent-subtle: #EFF6FF;       /* nền mục đang chọn, vùng thả hợp lệ */
  --color-accent-border: #BFDBFE;
  --color-accent-soft: #5B82E8;         /* chỉ dùng cho chuỗi "Dự kiến" trong biểu đồ */

  /* Trung tính */
  --color-page: #F8FAFC;                /* nền trang */
  --color-surface: #FFFFFF;             /* thẻ, bảng, sidebar, topbar */
  --color-subtle: #F1F5F9;              /* nền cột Kanban, header bảng, chip trung tính, skeleton nhạt */
  --color-line: #E2E8F0;                /* mọi viền 1px, skeleton đậm */
  --color-line-strong: #CBD5E1;         /* viền ô nhập, nút phụ, viền nét đứt */
  --color-ink: #0F172A;                 /* chữ chính, nền toast */
  --color-ink-2: #334155;               /* chữ trên chip nhãn xám */
  --color-ink-muted: #475569;           /* chữ phụ */
  --color-ink-faint: #64748B;           /* nhạt nhất được phép cho chữ */
  --color-disabled: #94A3B8;            /* chữ/icon bị vô hiệu hóa — không dùng cho chữ thường */

  /* Trạng thái: màu chữ / nền badge / viền badge */
  --color-ok: #15803D;      --color-ok-bg: #E8F6EC;      --color-ok-border: #BBE7C8;
  --color-warn: #B45309;    --color-warn-bg: #FEF3C7;    --color-warn-border: #FDE68A;
  --color-bad: #B91C1C;     --color-bad-bg: #FEE2E2;     --color-bad-border: #FECACA;
  --color-draft: #64748B;   --color-draft-bg: #F1F5F9;   --color-draft-border: #E2E8F0;  /* chữ badge dùng #475569 */
  --color-active: #0F766E;  --color-active-bg: #E0F7F4;  --color-active-border: #A7E0D7;

  /* Nhãn (label) — bảng riêng, KHÔNG mượn màu trạng thái */
  --color-label-a-bg: #F1F5F9; --color-label-a-fg: #334155;
  --color-label-b-bg: #FCE7F3; --color-label-b-fg: #9D174D;
  --color-label-c-bg: #F4F1EA; --color-label-c-fg: #57534E;

  /* Khối AI "Gợi ý" */
  --color-ai-bg: #FBF8F2;
  --color-ai-border: #E7DFD0;
  --color-ai-chip-bg: #F4F1EA;
  --color-ai-fg: #57534E;

  --radius-sm: 4px;   /* badge, chip, nút nhỏ trong khối AI, kbd */
  --radius-md: 6px;   /* thẻ, nút, ô nhập, panel, modal — TỐI ĐA */

  --shadow-overlay: 0 8px 24px rgba(15, 23, 42, 0.12);  /* CHỈ modal, dropdown, toast */
}
```

Font: tải Be Vietnam Pro **chỉ 400 và 600**. Không dùng 500/700.

## 2. Chữ và số

| Cỡ/độ đậm | Dùng cho |
|---|---|
| 20 / 600 | Tiêu đề trang (`h1`), số liệu lớn trên thẻ chỉ số |
| 14 / 600 | Tiêu đề khối, tên công việc, nút, mục menu đang chọn |
| 14 / 400 | Nội dung, ô bảng, mục menu |
| 12 / 600 | Badge, chip, tiêu đề cột bảng, nhãn ô nhập, nhóm menu |
| 12 / 400 | Phụ chú, meta ("Chu kỳ 24 · 03/11 – 16/11/2026") |

- **Mọi số liệu** (giờ, SP, %, đếm, WIP) có `tabular-nums` (tạo utility `.num`).
- **Định dạng số kiểu Việt Nam** qua một hàm duy nhất `shared/lib/format.ts` dùng `Intl.NumberFormat("vi-VN")`: thập phân dấu phẩy `14,5`, nghìn dấu chấm `5.480`. Không ghép chuỗi tay.
- Mẫu hiển thị chuẩn: giờ thực tế/dự kiến `14,5/16h`; Story Point `8 SP`; WIP `WIP 3/5`; điểm phần trăm `+2 đ%`; ngày `03/11/2026`; khoảng ngày `03/11 – 16/11/2026`.
- Phân cách meta bằng ` · `.

## 3. Màu — luật dùng

- **Màu nhấn** cho: nút chính, liên kết, mục đang chọn, viền focus, tab đang chọn (gạch chân 2px), vùng thả hợp lệ.
  Ngoại lệ đã có trong thiết kế: **thanh dữ liệu trung tính** (thanh tiến độ SP, "Giờ của tôi tuần này", biểu đồ cột một chuỗi) dùng `accent`; biểu đồ hai chuỗi: Dự kiến `accent-soft`, Thực tế `accent`.
- **Màu trạng thái chỉ để biểu thị trạng thái**, luôn đi kèm chữ. "Đang tiến hành" là xanh ngọc `active`, không bao giờ xanh dương.
- **Nhãn** dùng bảng label A/B/C, gán theo thứ tự hoặc hash tên nhãn. "Bug" là nhãn phân loại → màu nhãn; mức nghiêm trọng mới dùng màu trạng thái.
- **Avatar chữ cái** (2 ký tự) dùng bảng màu nhãn.
- CẤM: gradient, glassmorphism, tím, màu Tailwind mặc định (`blue-500`…), hex viết thẳng trong component.

## 4. Bố cục ứng dụng (app shell)

Tối ưu 1366×768, thiết kế ở 1440. Desktop-only.

**Sidebar trái** — rộng 236px, nền `surface`, viền phải 1px, padding 12/10.
- Logo: ô vuông 28px nền `accent` bo 6, icon đồng hồ trắng + "WorkTrack" (14/600) + vai trò hiện tại (12, `ink-faint`).
- Nhóm menu: tiêu đề nhóm 12/600 `ink-faint` ("Làm việc", "Theo dõi", "Quản trị"). Mục: cao 40px, icon 16px, 14px `ink-muted`; **đang chọn**: nền `accent-subtle`, chữ + icon `accent-hover`, 600.
- Menu đổi theo vai trò (ẩn mục không có quyền).
- Đáy: khối "Giờ của tôi tuần này" (`31,5/40h` + thanh 4px), rồi avatar 28px + tên + vai trò.

**Topbar** — nền `surface`, viền dưới 1px, padding 8/20.
- Trái: breadcrumb ("Dự án ›" 12px, tên dự án 14/600, badge trạng thái dự án).
- Phải: chỉ báo `● Đang cập nhật tức thời` (12, `ok`) · chồng avatar người đang xem (24px, chồng -7px, viền trắng 2px, "+n") · ô tìm 210px có phím `/` · nút chuông 34px, chấm đỏ 6px khi có thông báo chưa đọc.
- **Dải thông báo hệ thống nằm ngay dưới topbar**, không che nội dung (mất realtime: nền `warn-bg`; lưu thất bại: nền `bad-bg`).

**Đầu trang** — padding 12/20: `h1` 20/600 + meta 12px cùng hàng; bên phải: nút phụ (vd. "Phân rã bằng AI") + nút chính có kbd (`Tạo việc [N]`).
**Tab** dưới đầu trang: 14px, tab chọn chữ `accent` 600 + gạch chân 2px; bên phải cùng hàng: chip bộ lọc + chú giải màu.

## 5. Component chuẩn

| Component | Quy cách |
|---|---|
| **Nút chính** | nền `accent`, chữ trắng 14/600, bo 6, padding 9/14, cao tối thiểu **40px**, hover `accent-hover` |
| **Nút phụ** | nền trắng, viền `line-strong`, chữ `ink` |
| **Nút nhẹ** | nền trong suốt, chữ `ink-muted` ("Bỏ qua", "Hủy") |
| **Nút trạng thái** | "Nghiệm thu": nền `ok`; "Từ chối (kèm lý do)": nền trắng, chữ `bad`, viền `bad-border`; nút xác nhận trong hộp cảnh báo dùng màu theo mức (`warn`/`bad`) |
| **Nút vô hiệu** | nền `subtle`, chữ `disabled`, viền `line`, `cursor: not-allowed` + **lý do hiển thị ngay cạnh** |
| **Ô nhập** | nhãn 12/600 `ink-muted` ở trên; ô bo 6, viền `line-strong`, cao 40px; focus: viền 2px `accent`; lỗi: viền `bad` + dòng lỗi 12px `bad` bên dưới |
| **Badge trạng thái** | 12/600, chữ + nền + viền theo bộ trạng thái, bo 4, padding 2/8 |
| **Chip nhãn** | 12/600, nền/chữ label, bo 4, padding 1/6, không viền |
| **Chip bộ lọc** | đang áp dụng: nền `accent-subtle`, viền `accent-border`, chữ `accent-hover`, có "×"; chưa áp dụng: trắng, viền `line` |
| **Kbd** | 12/600, nền trắng, viền `line`, bo 4, padding 0/5 |
| **Bảng** | khung viền 1px bo 6; header nền `subtle`, 12/600 `ink-faint`; hàng 36–40px, ô 14px, phân cách viền trên 1px; cột số căn phải + `tabular-nums` |
| **Thẻ chỉ số** | nền trắng, viền, bo 6, padding 16: tiêu đề 12 `ink-faint` → số 20/600 → phụ chú 12. Xu hướng: tam giác + "±n đ%" màu theo tốt/xấu. Không thẻ KPI khổng lồ |
| **Thanh tiến độ** | cao 4px (thẻ) hoặc 6px (thẻ chỉ số), nền `line`, bo tròn |
| **Toast** | nền `ink`, chữ trắng 12px, icon ✓, bo 6, `shadow-overlay`, **góc dưới phải, 3 giây, một dòng**, nút "Hoàn tác" (chữ `accent-border`) khi đảo được. Không dùng toast cho lỗi cần đọc kỹ |
| **Hộp thoại** | nền trắng, viền, bo 6, `shadow-overlay`, padding 18; icon mức độ + tiêu đề 14/600 + mô tả 12; nút phải-dưới: phụ rồi chính |
| **Khối thông tin** | nền `subtle`, viền `line`, icon "i" — vd. "Màn hình này chỉ để theo dõi" trên Dashboard CEO |
| **Icon** | nét đơn (lucide-react), `strokeWidth={2}`, đầu tròn, 16px (menu, nút), 14–15px (trong nút nhỏ), 36px (trạng thái rỗng/lỗi). Không emoji |

## 6. Nét riêng của WorkTrack

**Thẻ công việc (một cách hiển thị ở mọi màn hình)** — nền trắng, viền 1px, bo 6, padding 10, gap 6:
1. Hàng 1: tên việc 14/600 (line-height 1.25) · chip `8 SP` bên phải (nền `subtle`).
2. Thanh so giờ 4px.
3. Hàng 3: ký hiệu mức + `14,5/16h · 2/3 con` (12, `ink-muted`) · avatar người nhận 20px bên phải.
- Cờ trễ hạn do AI và nhãn thêm vào hàng 3, không làm thẻ cao hơn → đảm bảo ≥ 6 thẻ/cột ở 1366×768.
- Công việc con: không có chip SP.

**Thanh so giờ — luôn kèm hình khối, không chỉ dựa vào màu**

| Mức | Điều kiện | Màu | Ký hiệu 8–9px |
|---|---|---|---|
| Trong định mức | thực tế ≤ dự kiến | `ok` | hình vuông |
| Vượt nhẹ | vượt dưới 20% | `warn` | hình tròn |
| Vượt nặng | vượt trên 20% | `bad` | tam giác |

Chú giải ba mức đặt trên thanh công cụ Kanban. Ngưỡng tính trong **một** hàm `getHoursLevel(actual, estimated)` dùng chung FE (và phải khớp backend).

**Kanban** — cột nền `subtle`, viền 1px, bo 6, padding 8, gap 6, rộng tối thiểu 200px, các cột cách 8px, cuộn ngang khi hẹp. Header cột: tên 14/600 + số lượng 12 + nút "+" 20px. Thẻ đang kéo để lại chỗ trống viền nét đứt `line-strong`, nền `page`.

**Kéo thả** — vùng hợp lệ: viền 2px `accent` + nền `accent-subtle` + chữ hướng dẫn. Vùng bị chặn: viền nét đứt, nền `subtle`, `opacity: .6`, icon khóa + **lý do cụ thể** ("Đỗ Thu Hà đã đủ 5 việc đang làm", "chỉ Quản lý dự án chuyển việc sang nhóm hoàn thành"). Không bao giờ chặn im lặng. Lưu thất bại → thẻ về cột cũ + dải lỗi dưới topbar.

**WIP** — chip `WIP 3/5` (viền `line-strong`); đủ giới hạn: `WIP 5/5` màu `bad`. Bị chặn nhận việc → hộp thoại "Không thể nhận thêm việc" có badge `5/5`, giải thích "Việc con không tính vào WIP", nút "Xem 5 việc đang làm".

**Khối AI "Gợi ý"** — nền `ai-bg`, viền `ai-border`, bo 6, padding 12. Chip "✦ Gợi ý" (nền `ai-chip-bg`, chữ `ai-fg`, icon lấp lánh). Ba nút cỡ nhỏ (12/600, bo 4): **Chấp nhận** (chính) · **Sửa** (phụ) · **Bỏ qua** (nhẹ). Hiển thị nhỏ thời gian phản hồi.
Đang xử lý (> 5 giây): khung viền `accent-border`, spinner + "Trợ lý AI đang phân rã yêu cầu…" (14/600 `accent-hover`), 3 skeleton 44px, câu "Bạn có thể tiếp tục làm việc khác", nút **Dừng**.

**Vai trò** — không có quyền → **ẩn hẳn** (vd. Member không thấy "Nghiệm thu"). Tạm thời không dùng được → vô hiệu hóa **kèm lý do**. Dashboard CEO: không có nút tạo/sửa/xóa, có khối thông tin "chỉ để theo dõi".

**Realtime** — `● Đang cập nhật tức thời` khi kết nối; mất kết nối → dải `warn`: "Mất kết nối cập nhật tức thời. Thay đổi của người khác sẽ không hiện ngay — đang kết nối lại…".

## 7. Trạng thái bắt buộc (artboard 13)

| Trạng thái | Quy cách |
|---|---|
| Đang tải | Skeleton **đúng kích thước thật** (thẻ 54px, hàng AI 44px), xen `line` và `subtle`; bố cục không nhảy khi dữ liệu về |
| Rỗng | Khung nét đứt `line-strong`, nền `page`, icon 36px `ink-faint`, tiêu đề 14/600, câu hướng dẫn 12 (tối đa ~260px), 1–2 nút hành động. Câu chữ theo vai trò (Manager: "Bạn chưa quản lý dự án nào" + Tạo dự án / Dùng mẫu; Member: "Backlog đang trống" + Xem công việc của tôi) |
| Lỗi tải | Như rỗng nhưng viền `bad-border`, icon `bad`, có mã lỗi tham chiếu + "Dữ liệu bạn đang xem có thể đã cũ", nút **Thử lại** + **Báo lỗi** |
| Không có quyền | Icon khóa, nêu lý do theo vai trò, nút về trang của tôi |
| Thành công | Toast (§5) |
| Cảnh báo nghiệp vụ | Hộp thoại: còn việc con chưa xong ("Để sau" / "Vẫn chuyển" màu `warn`); từ chối nghiệm thu (textarea lý do **bắt buộc** + checkbox "Trả thẻ về cột Đang làm và thông báo cho người nhận", nút "Từ chối" màu `bad`) |
| Log time sai | Giờ âm/0 → lỗi dưới ô "Không được log số giờ âm."; Done quá 24h → khối thông tin có icon khóa thay cho form |

## 8. Câu chữ, chuyển động, phím tắt, truy cập

- Tiếng Việt, ngắn, động từ rõ: "Nhận việc", "Log giờ", "Nghiệm thu", "Từ chối (kèm lý do)", "Tạo công việc mục tiêu", "Tạo công việc con", "Phân rã bằng AI".
- Thông báo lỗi nói **chuyện gì xảy ra + làm gì tiếp** ("Thẻ đã trở về cột cũ, vui lòng kéo lại").
- Dữ liệu mẫu: tên người Việt, dự án IT thật ("Cổng thông tin khách hàng", "Tích hợp cổng thanh toán VNPay").
- Chuyển động 120–200ms, chỉ để báo thay đổi trạng thái. Phím tắt: `N` tạo việc, `/` tìm kiếm (hiện dạng kbd).
- WCAG AA; vùng bấm cao tối thiểu 40px; mọi ô nhập có `<label>`; nút chỉ có icon có `aria-label`; biểu đồ SVG có `role="img"` + `aria-label` mô tả số liệu; ký hiệu hình khối cho mức giờ.

## 9. Bản đồ artboard → màn hình

| # | Artboard (file trong `docs/design/`) | Màn hình / feature | Vai trò |
|---|---|---|---|
| 01 | `artboards/01-design-foundation.html` — Nền tảng thiết kế | `shared/ui` | — |
| 13 | `artboards/13-screen-states.html` — Trạng thái & cảnh báo | `shared/ui` (EmptyState, ErrorState, Toast, Dialog, Banner) | Tất cả |
| 02 | `artboards/02-kanban-manager.html` — Bảng Kanban | `features/kanban` + app shell | Manager |
| 03 | `artboards/03-task-detail-logtime.html` — Chi tiết công việc + Log giờ | `features/tasks`, `features/timesheet` | Manager, Member |
| 04 | `artboards/04-member-board.html` — Bảng của Member (WIP 3/5) | `features/kanban` (biến thể Member), "Công việc của tôi" | Member |
| 05 | `artboards/05-ai-decompose.html` — AI phân rã yêu cầu | `features/ai` | Manager |
| 06 | `artboards/06-ai-assign-risk.html` — AI gợi ý nhân sự & cảnh báo trễ hạn | `features/ai` | Manager |
| 07 | `artboards/07-dashboard-project.html` — Dashboard dự án | `features/reports` | Manager |
| 08 | `artboards/08-performance-reports.html` — Báo cáo hiệu suất | `features/reports` | Manager |
| 09 | `artboards/09-dashboard-company-ceo.html` — Dashboard toàn công ty (chỉ xem) | `features/reports` | CEO |
| 10 | `artboards/10-admin-users-departments.html` — Nhân sự & phòng ban | `features/admin` | Admin |
| 11 | `artboards/11-import-employees.html` — Import nhân sự từ Excel | `features/admin` (Mở rộng) | Admin |
| 12 | `artboards/12-project-settings.html` — Cấu hình dự án & Kanban | `features/projects` | Manager |

Artboard dùng inline style và dữ liệu cứng: **chép bố cục, khoảng cách, câu chữ — không chép HTML**. Dựng lại bằng component `shared/ui` + token.

## 10. Điểm lệch trong bản thiết kế — đã quy ước

| Điểm lệch | Quy ước trong code |
|---|---|
| Nút trong artboard 13 cao 44px, nền tảng ghi 40px | Dùng **40px** mọi nơi |
| Artboard 13 tải font 500/700 | Chỉ **400/600** |
| Nguyên tắc "màu nhấn chỉ cho nút/liên kết…" nhưng biểu đồ, thanh tiến độ dùng màu nhấn | Cho phép cho thanh dữ liệu trung tính (§3) |
| Đề bài cũ ghi biểu đồ Dự kiến xanh / Thực tế cam | Theo thiết kế: Dự kiến `#5B82E8`, Thực tế `#1D4ED8`; số liệu bên phải tô theo mức giờ |
| Badge trạng thái bo 4px, chip đếm trạng thái trên Dashboard CEO bo 6px | Badge/chip đều **4px** |
| Ngưỡng đúng 20% chưa rõ thuộc vàng hay đỏ | ⚠ **Chưa chốt** — hỏi người dùng; tạm `< 20%` vàng, `≥ 20%` đỏ |
| Mã lỗi hiển thị dạng "WT-5031", API dùng mã chữ (`TASK_WIP_LIMIT`) | ⚠ **Chưa chốt** — xem `CLAUDE.md` §6 |
