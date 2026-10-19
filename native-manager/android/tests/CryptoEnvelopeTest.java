package com.codex.u30vpn;

import java.nio.charset.StandardCharsets;
import java.security.KeyPair;
import java.security.KeyPairGenerator;
import java.security.SecureRandom;
import java.security.spec.MGF1ParameterSpec;
import java.util.Base64;
import java.util.regex.Matcher;
import java.util.regex.Pattern;
import javax.crypto.Cipher;
import javax.crypto.spec.GCMParameterSpec;
import javax.crypto.spec.OAEPParameterSpec;
import javax.crypto.spec.PSource;
import javax.crypto.spec.SecretKeySpec;

/** Host-only: exercises real JCA crypto, independent of Android stubs. */
public final class CryptoEnvelopeTest {
    private static final byte[] LABEL = "u30pro-vpn-web-v1".getBytes(StandardCharsets.US_ASCII);

    public static void main(String[] args) throws Exception {
        KeyPairGenerator generator = KeyPairGenerator.getInstance("RSA");
        generator.initialize(2048, new SecureRandom());
        KeyPair pair = generator.generateKeyPair();
        String pem = "-----BEGIN PUBLIC KEY-----\n"
                + Base64.getMimeEncoder(64, new byte[]{'\n'}).encodeToString(pair.getPublic().getEncoded())
                + "\n-----END PUBLIC KEY-----\n";
        String plain = "{\"v\":1,\"issuedAt\":1800000000000,\"nonce\":\"AAAAAAAAAAAAAAAAAAAAAA\","
                + "\"action\":\"select\",\"id\":\"n-0123456789abcdef\"}";
        String first = CryptoEnvelope.seal(pem, plain);
        String second = CryptoEnvelope.seal(pem, plain);
        check(!first.equals(second), "encryption must use fresh key and IV");
        String envelope = new String(Base64.getUrlDecoder().decode(first), StandardCharsets.UTF_8);
        check(envelope.startsWith("{\"v\":1,"), "versioned JSON envelope");
        byte[] wrapped = field(envelope, "key");
        byte[] iv = field(envelope, "iv");
        byte[] data = field(envelope, "data");
        check(wrapped.length == 256 && iv.length == 12, "RSA2048 and GCM IV sizes");
        Cipher rsa = Cipher.getInstance("RSA/ECB/OAEPPadding");
        rsa.init(Cipher.DECRYPT_MODE, pair.getPrivate(), new OAEPParameterSpec(
                "SHA-256", "MGF1", MGF1ParameterSpec.SHA256, new PSource.PSpecified(LABEL)));
        byte[] key = rsa.doFinal(wrapped);
        check(key.length == 32, "AES256 key");
        Cipher aes = Cipher.getInstance("AES/GCM/NoPadding");
        aes.init(Cipher.DECRYPT_MODE, new SecretKeySpec(key, "AES"), new GCMParameterSpec(128, iv));
        aes.updateAAD(LABEL);
        check(plain.equals(new String(aes.doFinal(data), StandardCharsets.UTF_8)), "plaintext roundtrip");
        data[data.length - 1] ^= 1;
        boolean rejected = false;
        try {
            aes.init(Cipher.DECRYPT_MODE, new SecretKeySpec(key, "AES"), new GCMParameterSpec(128, iv));
            aes.updateAAD(LABEL);
            aes.doFinal(data);
        } catch (java.security.GeneralSecurityException expected) { rejected = true; }
        check(rejected, "tampered GCM tag must be rejected");
        rejected = false;
        try { CryptoEnvelope.seal("not a public key", plain); }
        catch (java.security.GeneralSecurityException | IllegalArgumentException expected) { rejected = true; }
        check(rejected, "invalid key must not be accepted");
        if (args.length == 1 && args[0].equals("--fixture")) {
            System.out.println("{\"privateKeyPkcs8\":\""
                    + Base64.getEncoder().encodeToString(pair.getPrivate().getEncoded())
                    + "\",\"envelope\":\"" + first + "\"}");
        } else {
            System.out.println("CRYPTO_ENVELOPE_TEST_OK: RSA-OAEP-SHA256/MGF1-SHA256 label, AES256-GCM AAD, randomization, tamper rejection");
        }
    }

    private static byte[] field(String json, String name) {
        Matcher matcher = Pattern.compile("\\\"" + name + "\\\":\\\"([A-Za-z0-9+/=]+)\\\"").matcher(json);
        if (!matcher.find()) throw new AssertionError("missing field: " + name);
        return Base64.getDecoder().decode(matcher.group(1));
    }

    private static void check(boolean valid, String message) {
        if (!valid) throw new AssertionError(message);
    }
}
