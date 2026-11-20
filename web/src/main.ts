import { createApp } from 'vue'
import { createPinia } from 'pinia'
import { loader } from '@guolao/vue-monaco-editor'
import * as monaco from 'monaco-editor'
// monaco 0.57 exports 映射：monaco-editor/editor/* → esm/vs/editor/*
import editorWorker from 'monaco-editor/editor/editor.worker?worker'
import App from './App.vue'
import router from './router'
import './styles/index.css'

// Monaco 从本地包加载，不依赖外网 CDN
self.MonacoEnvironment = {
  getWorker() {
    return new editorWorker()
  },
}
loader.config({ monaco })

const app = createApp(App)
app.use(createPinia())
app.use(router)
app.mount('#app')
