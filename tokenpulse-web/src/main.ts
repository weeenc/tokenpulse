import { createApp } from 'vue';
import { createPinia } from 'pinia';
import './styles.css';
import App from './App.vue';
import router from './router/index.js';
import spotlight from './directives/spotlight.js';

createApp(App).directive('spotlight', spotlight).use(createPinia()).use(router).mount('#app');
