// vite.config.js
import { defineConfig } from "file:///home/ruben/projects/2025/0211-opensoho/ui/node_modules/vite/dist/node/index.js";
import { svelte, vitePreprocess } from "file:///home/ruben/projects/2025/0211-opensoho/ui/node_modules/@sveltejs/vite-plugin-svelte/src/index.js";
var __vite_injected_original_dirname = "/home/ruben/projects/2025/0211-opensoho/ui";
var vite_config_default = defineConfig({
  server: {
    port: 3e3
  },
  envPrefix: "PB",
  base: "./",
  build: {
    chunkSizeWarningLimit: 1e3,
    reportCompressedSize: false
  },
  plugins: [
    svelte({
      preprocess: [vitePreprocess()],
      onwarn: (warning, handler) => {
        if (warning.code.startsWith("a11y-")) {
          return;
        }
        handler(warning);
      }
    })
  ],
  resolve: {
    alias: {
      "@": __vite_injected_original_dirname + "/src"
    }
  }
});
export {
  vite_config_default as default
};
//# sourceMappingURL=data:application/json;base64,ewogICJ2ZXJzaW9uIjogMywKICAic291cmNlcyI6IFsidml0ZS5jb25maWcuanMiXSwKICAic291cmNlc0NvbnRlbnQiOiBbImNvbnN0IF9fdml0ZV9pbmplY3RlZF9vcmlnaW5hbF9kaXJuYW1lID0gXCIvaG9tZS9ydWJlbi9wcm9qZWN0cy8yMDI1LzAyMTEtb3BlbnNvaG8vdWlcIjtjb25zdCBfX3ZpdGVfaW5qZWN0ZWRfb3JpZ2luYWxfZmlsZW5hbWUgPSBcIi9ob21lL3J1YmVuL3Byb2plY3RzLzIwMjUvMDIxMS1vcGVuc29oby91aS92aXRlLmNvbmZpZy5qc1wiO2NvbnN0IF9fdml0ZV9pbmplY3RlZF9vcmlnaW5hbF9pbXBvcnRfbWV0YV91cmwgPSBcImZpbGU6Ly8vaG9tZS9ydWJlbi9wcm9qZWN0cy8yMDI1LzAyMTEtb3BlbnNvaG8vdWkvdml0ZS5jb25maWcuanNcIjtpbXBvcnQgeyBkZWZpbmVDb25maWcgfSAgICAgICAgICAgZnJvbSAndml0ZSc7XG5pbXBvcnQgeyBzdmVsdGUsIHZpdGVQcmVwcm9jZXNzIH0gZnJvbSAnQHN2ZWx0ZWpzL3ZpdGUtcGx1Z2luLXN2ZWx0ZSc7XG5cbi8vIHNlZSBodHRwczovL3ZpdGVqcy5kZXYvY29uZmlnXG5leHBvcnQgZGVmYXVsdCBkZWZpbmVDb25maWcoe1xuICAgIHNlcnZlcjoge1xuICAgICAgICBwb3J0OiAzMDAwLFxuICAgIH0sXG4gICAgZW52UHJlZml4OiAnUEInLFxuICAgIGJhc2U6ICcuLycsXG4gICAgYnVpbGQ6IHtcbiAgICAgICAgY2h1bmtTaXplV2FybmluZ0xpbWl0OiAxMDAwLFxuICAgICAgICByZXBvcnRDb21wcmVzc2VkU2l6ZTogZmFsc2UsXG4gICAgfSxcbiAgICBwbHVnaW5zOiBbXG4gICAgICAgIHN2ZWx0ZSh7XG4gICAgICAgICAgICBwcmVwcm9jZXNzOiBbdml0ZVByZXByb2Nlc3MoKV0sXG4gICAgICAgICAgICBvbndhcm46ICh3YXJuaW5nLCBoYW5kbGVyKSA9PiB7XG4gICAgICAgICAgICAgICAgaWYgKHdhcm5pbmcuY29kZS5zdGFydHNXaXRoKCdhMTF5LScpKSB7XG4gICAgICAgICAgICAgICAgICAgIHJldHVybjsgLy8gc2lsZW5jZSBhMTF5IHdhcm5pbmdzXG4gICAgICAgICAgICAgICAgfVxuICAgICAgICAgICAgICAgIGhhbmRsZXIod2FybmluZyk7XG4gICAgICAgICAgICB9LFxuICAgICAgICB9KSxcbiAgICBdLFxuICAgIHJlc29sdmU6IHtcbiAgICAgICAgYWxpYXM6IHtcbiAgICAgICAgICAgICdAJzogX19kaXJuYW1lICsgJy9zcmMnLFxuICAgICAgICB9XG4gICAgfSxcbn0pXG4iXSwKICAibWFwcGluZ3MiOiAiO0FBQWdULFNBQVMsb0JBQThCO0FBQ3ZWLFNBQVMsUUFBUSxzQkFBc0I7QUFEdkMsSUFBTSxtQ0FBbUM7QUFJekMsSUFBTyxzQkFBUSxhQUFhO0FBQUEsRUFDeEIsUUFBUTtBQUFBLElBQ0osTUFBTTtBQUFBLEVBQ1Y7QUFBQSxFQUNBLFdBQVc7QUFBQSxFQUNYLE1BQU07QUFBQSxFQUNOLE9BQU87QUFBQSxJQUNILHVCQUF1QjtBQUFBLElBQ3ZCLHNCQUFzQjtBQUFBLEVBQzFCO0FBQUEsRUFDQSxTQUFTO0FBQUEsSUFDTCxPQUFPO0FBQUEsTUFDSCxZQUFZLENBQUMsZUFBZSxDQUFDO0FBQUEsTUFDN0IsUUFBUSxDQUFDLFNBQVMsWUFBWTtBQUMxQixZQUFJLFFBQVEsS0FBSyxXQUFXLE9BQU8sR0FBRztBQUNsQztBQUFBLFFBQ0o7QUFDQSxnQkFBUSxPQUFPO0FBQUEsTUFDbkI7QUFBQSxJQUNKLENBQUM7QUFBQSxFQUNMO0FBQUEsRUFDQSxTQUFTO0FBQUEsSUFDTCxPQUFPO0FBQUEsTUFDSCxLQUFLLG1DQUFZO0FBQUEsSUFDckI7QUFBQSxFQUNKO0FBQ0osQ0FBQzsiLAogICJuYW1lcyI6IFtdCn0K
