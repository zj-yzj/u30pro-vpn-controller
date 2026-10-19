package com.codex.u30vpn;

import java.nio.charset.StandardCharsets;
import java.security.GeneralSecurityException;
import java.security.KeyFactory;
import java.security.PublicKey;
import java.security.SecureRandom;
import java.security.interfaces.RSAPublicKey;
import java.security.spec.MGF1ParameterSpec;
import java.security.spec.X509EncodedKeySpec;
import java.util.Arrays;
import java.util.Base64;
import javax.crypto.Cipher;
import javax.crypto.spec.GCMParameterSpec;
import javax.crypto.spec.OAEPParameterSpec;
import javax.crypto.spec.PSource;
import javax.crypto.spec.SecretKeySpec;

public final class CryptoEnvelope {
    private static final byte[] LABEL = "u30pro-vpn-web-v1".getBytes(StandardCharsets.US_ASCII);
    private static final SecureRandom RANDOM = new SecureRandom();
    private CryptoEnvelope() {}

    public static String seal(String publicKeyPem, String plaintext) throws GeneralSecurityException {
        if (publicKeyPem == null || publicKeyPem.length() > 4096
                || !publicKeyPem.trim().startsWith("-----BEGIN PUBLIC KEY-----")
                || !publicKeyPem.trim().endsWith("-----END PUBLIC KEY-----")) {
            throw new GeneralSecurityException("invalid public key");
        }
        byte[] plain = plaintext.getBytes(StandardCharsets.UTF_8);
        if (plain.length == 0 || plain.length > 8192) throw new GeneralSecurityException("invalid request length");
        String body = publicKeyPem.replace("-----BEGIN PUBLIC KEY-----", "")
                .replace("-----END PUBLIC KEY-----", "").replaceAll("\\s", "");
        PublicKey publicKey = KeyFactory.getInstance("RSA")
                .generatePublic(new X509EncodedKeySpec(Base64.getDecoder().decode(body)));
        if (!(publicKey instanceof RSAPublicKey) || ((RSAPublicKey) publicKey).getModulus().bitLength() != 2048) {
            throw new GeneralSecurityException("invalid public key size");
        }
        byte[] key = new byte[32];
        byte[] iv = new byte[12];
        RANDOM.nextBytes(key);
        RANDOM.nextBytes(iv);
        try {
            Cipher aes = Cipher.getInstance("AES/GCM/NoPadding");
            aes.init(Cipher.ENCRYPT_MODE, new SecretKeySpec(key, "AES"), new GCMParameterSpec(128, iv));
            aes.updateAAD(LABEL);
            byte[] encrypted = aes.doFinal(plain);
            Cipher rsa = Cipher.getInstance("RSA/ECB/OAEPPadding");
            rsa.init(Cipher.ENCRYPT_MODE, publicKey, new OAEPParameterSpec(
                    "SHA-256", "MGF1", MGF1ParameterSpec.SHA256, new PSource.PSpecified(LABEL)), RANDOM);
            byte[] wrapped = rsa.doFinal(key);
            String json = "{\"v\":1,\"key\":\"" + Base64.getEncoder().encodeToString(wrapped)
                    + "\",\"iv\":\"" + Base64.getEncoder().encodeToString(iv)
                    + "\",\"data\":\"" + Base64.getEncoder().encodeToString(encrypted) + "\"}";
            return Base64.getUrlEncoder().withoutPadding().encodeToString(json.getBytes(StandardCharsets.UTF_8));
        } finally {
            Arrays.fill(key, (byte) 0);
            Arrays.fill(plain, (byte) 0);
        }
    }
}
