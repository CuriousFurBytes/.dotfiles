return {
    "catppuccin/nvim",
    lazy = false,
    name = "catppuccin",
    priority = 1000,
    config = function()
        flavour = "moccha"
        transparent_background = true
        term_colors = true
        -- auto_integrations = true
        vim.cmd.colorscheme("catppuccin")
    end
}
