/** @type {import('tailwindcss').Config} */
module.exports = {
    content: [
        "./views/components/**/*.templ",
        "./views/pages/**/*.templ",
        "./views/layouts/**/*.templ",
    ],
    theme: {
        extend: {},
    },
    plugins: [
        require("./scripts/node_modules/@tailwindcss/typography"),
        require("./scripts/node_modules/@tailwindcss/forms"),
        require("./scripts/node_modules/daisyui"),
    ],
};
