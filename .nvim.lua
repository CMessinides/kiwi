vim.filetype.add({
	extension = {
		ast = "astsexp",
		tmpl = "gotmpl",
	},
})

local augroup = vim.api.nvim_create_augroup("AstSexpSettings", {
	clear = true,
})

vim.api.nvim_create_autocmd("FileType", {
	group = augroup,
	pattern = "astsexp",
	callback = function()
		vim.opt_local.tabstop = 4
		vim.opt_local.shiftwidth = 4
		vim.opt_local.expandtab = false
	end,
})
