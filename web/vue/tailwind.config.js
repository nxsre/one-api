/** @type {import('tailwindcss').Config} */
export default {
  // Scope Tailwind so its reset/utilities never fight Ant Design Vue's own
  // component styles; we only opt-in to utilities, keeping preflight minimal.
  content: ['./index.html', './src/**/*.{vue,js,ts,jsx,tsx}'],
  corePlugins: {
    // Ant Design Vue manages its own base styles; disabling preflight avoids
    // clobbering antd typography/forms while keeping utility classes.
    preflight: false,
  },
  theme: {
    extend: {
      colors: {
        primary: '#1677ff',
      },
    },
  },
  plugins: [],
};
