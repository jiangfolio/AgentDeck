/// <reference types="vite/client" />

declare module 'sm-crypto' {
  export const sm2: {
    doEncrypt: (msg: string, publicKey: string, cipherMode?: number) => string
    doDecrypt: (encryptData: string, privateKey: string, cipherMode?: number) => string
    generateKeyPairHex: () => { publicKey: string; privateKey: string }
  }
  export const sm3: (str: string) => string
  export const sm4: {
    encrypt: (inArray: string | number[], key: string | number[]) => string
    decrypt: (inArray: string | number[], key: string | number[]) => string
  }
}
