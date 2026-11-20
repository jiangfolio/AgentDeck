import axios, { type AxiosRequestConfig } from 'axios'
import {
  clearPublicKeyCache,
  encryptSM4Key,
  ensurePublicKey,
  randomSM4KeyHex,
  sm4Decrypt,
  sm4Encrypt,
} from '@/utils/crypto'

export type ApiBody<T = any> = {
  code: number
  message: string
  data: T
}

const http = axios.create({
  baseURL: '/api',
  timeout: 30000,
  withCredentials: true,
})

http.interceptors.response.use(
  (res) => {
    const body = res.data as ApiBody
    if (body && typeof body.code === 'number' && body.code !== 0) {
      return Promise.reject(new Error(body.message || '请求失败'))
    }
    return res
  },
  (err) => Promise.reject(err),
)

/** 日常 API：明文 JSON。 */
export async function apiGet<T = any>(url: string, config?: AxiosRequestConfig) {
  const res = await http.get<ApiBody<T>>(url, config)
  return res.data.data
}

/** 日常 API：明文 JSON。 */
export async function apiPost<T = any>(url: string, data?: any, config?: AxiosRequestConfig) {
  const res = await http.post<ApiBody<T>>(url, data, config)
  return res.data.data
}

async function fetchPublicKey(): Promise<string> {
  const res = await axios.get<ApiBody<{ public_key: string }>>('/api/crypto/public-key', {
    withCredentials: true,
  })
  if (res.data.code !== 0 || !res.data.data?.public_key) {
    throw new Error(res.data.message || '获取公钥失败')
  }
  return res.data.data.public_key
}

/**
 * 登录专用：SM2 协商 SM4 后加密请求体，并解密响应体。
 * 仅登录走加密，其它接口请用 apiGet / apiPost。
 */
export async function encryptedPost<T = any>(url: string, data: any) {
  // 1. 取公钥并生成随机会话密钥
  const publicKey = await ensurePublicKey(fetchPublicKey)
  const sm4Key = randomSM4KeyHex()
  const plain = JSON.stringify(data)
  const cipher = sm4Encrypt(plain, sm4Key)

  // 2. 发送加密登录请求
  let res
  try {
    res = await http.post(url, { data: cipher }, {
      headers: { 'X-SM4-Key': encryptSM4Key(sm4Key, publicKey) },
    })
  } catch (e) {
    clearPublicKeyCache()
    throw e
  }

  // 3. 解密响应：形如 { data: "hex..." }；明文业务 JSON 则直接取 data
  const payload = res.data
  if (payload && typeof payload.data === 'string' && payload.code === undefined) {
    let decrypted: ApiBody<T>
    try {
      decrypted = JSON.parse(sm4Decrypt(payload.data, sm4Key)) as ApiBody<T>
    } catch {
      clearPublicKeyCache()
      throw new Error('解密响应失败')
    }
    if (decrypted.code !== 0) {
      throw new Error(decrypted.message || '请求失败')
    }
    return decrypted.data
  }

  const body = payload as ApiBody<T>
  if (body && typeof body.code === 'number' && body.code !== 0) {
    throw new Error(body.message || '请求失败')
  }
  return body.data
}

export default http
