interface Quote {
  text: string;
  author: string;
  rating: number; // 1-5 rating for impact/coolness
}

export const hackerQuotes: Quote[] = [
  {
    text: "Privacy is not an option, and it shouldn't be the price we accept for just getting on the Internet.",
    author: "Gary Kovacs",
    rating: 5,
  },
  {
    text: "If you think technology can solve your security problems, then you don't understand the problems and you don't understand the technology.",
    author: "Bruce Schneier",
    rating: 5,
  },
  {
    text: "The only truly secure system is one that is powered off, cast in a block of concrete and sealed in a lead-lined room with armed guards.",
    author: "Gene Spafford",
    rating: 4,
  },
  {
    text: "Security is always excessive until it's not enough.",
    author: "Robbie Sinclair",
    rating: 4,
  },
  {
    text: "In a world full of vulnerabilities, hackers are the immune system.",
    author: "Anonymous",
    rating: 3,
  },
  {
    text: "Digital security is like a chain: it's only as strong as its weakest link.",
    author: "Unknown",
    rating: 3,
  },
  {
    text: "There are two types of companies: those that have been hacked, and those who don't know they have been hacked.",
    author: "John Chambers",
    rating: 5,
  },
  {
    text: "The quieter you become, the more you are able to hear.",
    author: "Kali Linux",
    rating: 4,
  },
  {
    text: "Every application has at least two bugs: one is known, and one is unknown.",
    author: "Security Proverb",
    rating: 3,
  },
  {
    text: "The best way to predict the future is to create it.",
    author: "Peter Drucker",
    rating: 4,
  },
  {
    text: "The Internet was not designed with security in mind; it was designed with scalability in mind.",
    author: "Dan Kaminsky",
    rating: 5,
  },
  {
    text: "In the end, a penetration tester is trying to emulate what a real attacker would do.",
    author: "Kevin Mitnick",
    rating: 4,
  },
  {
    text: "The goal of security is not to build systems that are theoretically secure, but to build systems that are practically impossible to penetrate.",
    author: "James Gosling",
    rating: 5,
  },
  {
    text: "Passwords are like underwear: don't let people see it, change it very often, and you shouldn't share it with strangers.",
    author: "Security Humor",
    rating: 3,
  },
  {
    text: "The difference between a script kiddie and a professional penetration tester is the permission slip.",
    author: "Unknown",
    rating: 3,
  },
];
