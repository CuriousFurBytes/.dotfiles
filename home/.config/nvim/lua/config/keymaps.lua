local opts = {
    noremap = true,
    silent = true
}

vim.keymap.set({"n", "i"}, "<C-s>", "<cmd>w<cr>", opts) -- Save file

vim.keymap.set({"n", "i"}, "<C-n>", ":tabnew<cr>", opts) -- Create new tab
