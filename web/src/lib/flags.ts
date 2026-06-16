// Country name → emoji flag mapping for WC2026 nations.
export const FLAGS: Record<string, string> = {
  Mexico: '🇲🇽', 'South Korea': '🇰🇷', Norway: '🇳🇴', Ecuador: '🇪🇨',
  Canada: '🇨🇦', Japan: '🇯🇵', Senegal: '🇸🇳', Paraguay: '🇵🇾',
  USA: '🇺🇸', Switzerland: '🇨🇭', 'Ivory Coast': '🇨🇮', Uzbekistan: '🇺🇿',
  Argentina: '🇦🇷', Australia: '🇦🇺', Egypt: '🇪🇬', Scotland: '🏴󠁧󠁢󠁳󠁣󠁴󠁿',
  France: '🇫🇷', Croatia: '🇭🇷', Morocco: '🇲🇦', Qatar: '🇶🇦',
  Brazil: '🇧🇷', Netherlands: '🇳🇱', Tunisia: '🇹🇳', Jordan: '🇯🇴',
  England: '🏴󠁧󠁢󠁥󠁮󠁧󠁿', Colombia: '🇨🇴', Nigeria: '🇳🇬', 'New Zealand': '🇳🇿',
  Spain: '🇪🇸', Uruguay: '🇺🇾', Ghana: '🇬🇭', 'Saudi Arabia': '🇸🇦',
  Portugal: '🇵🇹', Belgium: '🇧🇪', Algeria: '🇩🇿', Panama: '🇵🇦',
  Germany: '🇩🇪', Iran: '🇮🇷', Cameroon: '🇨🇲', 'Costa Rica': '🇨🇷',
  Italy: '🇮🇹', Denmark: '🇩🇰', Mali: '🇲🇱', Honduras: '🇭🇳',
  Poland: '🇵🇱', Serbia: '🇷🇸', 'South Africa': '🇿🇦', Curacao: '🇨🇼',
}

export function flag(country: string): string {
  return FLAGS[country] ?? '🏳️'
}
