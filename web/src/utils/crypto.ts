import { sm2, sm4 } from 'sm-crypto'

let cachedPublicKey = ''

/** 拉取并缓存服务端 SM2 公钥（明文接口）。 */
export async function ensurePublicKey(fetcher: () => Promise<string>): Promise<string> {
  if (cachedPublicKey) return cachedPublicKey
  cachedPublicKey = await fetcher()
  return cachedPublicKey
}

export function clearPublicKeyCache() {
  cachedPublicKey = ''
}

/** 生成 16 字节 SM4 密钥的十六进制表示。 */
export function randomSM4KeyHex(): string {
  const bytes = new Uint8Array(16)
  crypto.getRandomValues(bytes)
  return Array.from(bytes)
    .map((b) => b.toString(16).padStart(2, '0'))
    .join('')
}

/** 用 SM2 公钥加密 SM4 密钥（明文为 32 位 hex 字符串）。 */
export function encryptSM4Key(sm4KeyHex: string, publicKey: string): string {
  // sm-crypto 要求公钥带 04 前缀（130 hex）；cipherMode 1 = C1C3C2
  const pub = publicKey.startsWith('04') ? publicKey : `04${publicKey}`
  return sm2.doEncrypt(sm4KeyHex, pub, 1)
}

/** SM4-ECB 加密，返回 hex（不传 mode 时 sm-crypto 默认 ECB）。 */
export function sm4Encrypt(plain: string, keyHex: string): string {
  return sm4.encrypt(plain, keyHex)
}

/** SM4-ECB 解密。 */
export function sm4Decrypt(cipherHex: string, keyHex: string): string {
  return sm4.decrypt(cipherHex, keyHex)
}
