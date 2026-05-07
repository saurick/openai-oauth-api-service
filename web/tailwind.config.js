const formsPlugin = require('@tailwindcss/forms')
const typographyPlugin = require('@tailwindcss/typography')
const aspectRatioPlugin = require('@tailwindcss/aspect-ratio')

/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    './src/**/*.{js,jsx,ts,tsx}', // 扫描src目录下的所有文件
  ],
  theme: {
    extend: {}, // 扩展主题
  },
  plugins: [
    formsPlugin, // 常用表单样式优化
    typographyPlugin, // 文章内容优化（prose）
    aspectRatioPlugin, // 支持 aspect-ratio 工具
  ],
}
