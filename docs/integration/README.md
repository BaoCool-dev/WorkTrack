# docs/integration — báo cáo tích hợp từng bước

Mỗi bước gộp một nhánh vào nhánh tích hợp (mặc định `release/v1`) có một báo cáo `NN-<nhánh>.md`: test sau gộp và sau dọn, rà cấu hình, code rác đã dọn hoặc giữ lại, việc người dùng tự kiểm.
Tag cục bộ `integ/NN-<nhánh>` đánh dấu từng bước: `git diff integ/03-auth integ/04-org` xem đúng thay đổi của bước 4.
