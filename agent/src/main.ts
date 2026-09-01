import { createApp } from 'vue';
import App from './App.vue';
import { setupRouter } from './router';
import './styles/css/global.css';

async function bootstrap() {
  const app = createApp(App);
  await setupRouter(app);
  app.mount('#app');
}

void bootstrap();
