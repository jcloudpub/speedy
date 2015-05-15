#ifndef __SPY_ATOMIC_H__
#define __SPY_ATOMIC_H__

#define LOCK "lock ; "

/*
 * Make sure gcc doesn't try to be clever and move things around
 * on us. We need to use _exactly_ the address the user gave us,
 * not some alias that contains the same information.
 */
typedef struct { volatile int counter; } spy_atomic_t;

#define SPY_ATOMIC_INIT(i)	{ (i) }

/**
 * atomic_read - read atomic variable
 * @param v pointer of type spy_atomic_t
 * 
 * Atomically reads the value of v.
 */ 
#define spy_atomic_read(v)		((v)->counter)

/**
 * atomic_set - set atomic variable
 * @param v pointer of type spy_atomic_t
 * @param i required value
 * 
 * Atomically sets the value of v to i.
 */ 
#define spy_atomic_set(v,i)		(((v)->counter) = (i))

/* These are x86-specific, used by some header files */
#define spy_atomic_clear_mask(mask, addr) \
__asm__ __volatile__(LOCK "andl %0,%1" \
: : "r" (~(mask)),"m" (*addr) : "memory")

#define spy_atomic_set_mask(mask, addr) \
__asm__ __volatile__(LOCK "orl %0,%1" \
: : "r" (mask),"m" (*(addr)) : "memory")

#define spy_atomic_inc_return(v)  (spy_atomic_add_return(1,v))
#define spy_atomic_dec_return(v)  (spy_atomic_sub_return(1,v))

/**
 * atomic_add - add integer to atomic variable
 * @param i integer value to add
 * @param v pointer of type spy_atomic_t
 * 
 * Atomically adds i to v.
 */
static inline void spy_atomic_add(int i, spy_atomic_t *v)
{
	__asm__ __volatile__(
		LOCK "addl %1,%0"
		:"=m" (v->counter)
		:"ir" (i), "m" (v->counter));
}

/**
 * atomic_sub - subtract the atomic variable
 * @param i integer value to subtract
 * @param v pointer of type spy_atomic_t
 * 
 * Atomically subtracts i from v.
 */
static inline void spy_atomic_sub(int i, spy_atomic_t *v)
{
	__asm__ __volatile__(
		LOCK "subl %1,%0"
		:"=m" (v->counter)
		:"ir" (i), "m" (v->counter));
}


/**
 * atomic_add_return - add and return
 * @param v pointer of type spy_atomic_t
 * @param i integer value to add
 *
 * Atomically adds i to v and returns i + v
 */
static inline int spy_atomic_add_return(int i, spy_atomic_t *v)
{
  int __i;
  /* Modern 486+ processor */
  __i = i;
  __asm__ __volatile__(
                LOCK "xaddl %0, %1"
                :"+r" (i), "+m" (v->counter)
                : : "memory");
  return i + __i;
}

static inline int spy_atomic_sub_return(int i, spy_atomic_t *v)
{
  return spy_atomic_add_return(-i,v);
}


/**
 * atomic_sub_and_test - subtract value from variable and test result
 * @param i integer value to subtract
 * @param v pointer of type spy_atomic_t
 * 
 * Atomically subtracts i from v and returns
 * true if the result is zero, or false for all
 * other cases.
 */
static inline int spy_atomic_sub_and_test(int i, spy_atomic_t *v)
{
	unsigned char c;

	__asm__ __volatile__(
		LOCK "subl %2,%0; sete %1"
		:"=m" (v->counter), "=qm" (c)
		:"ir" (i), "m" (v->counter) : "memory");
	return c;
}

/**
 * atomic_inc - increment atomic variable
 * @param v pointer of type spy_atomic_t
 * 
 * Atomically increments v by 1.
 */ 
static inline void spy_atomic_inc(spy_atomic_t *v)
{
	__asm__ __volatile__(
		LOCK "incl %0"
		:"=m" (v->counter)
		:"m" (v->counter));
}

/**
 * atomic_dec - decrement atomic variable
 * @param v pointer of type spy_atomic_t
 * 
 * Atomically decrements v by 1.
 */ 
static inline void spy_atomic_dec(spy_atomic_t *v)
{
	__asm__ __volatile__(
		LOCK "decl %0"
		:"=m" (v->counter)
		:"m" (v->counter));
}

/**
 * atomic_dec_and_test - decrement and test
 * @param v pointer of type spy_atomic_t
 * 
 * Atomically decrements v by 1 and
 * returns true if the result is 0, or false for all other
 * cases.
 */ 
static inline int spy_atomic_dec_and_test(spy_atomic_t *v)
{
	unsigned char c;

	__asm__ __volatile__(
		LOCK "decl %0; sete %1"
		:"=m" (v->counter), "=qm" (c)
		:"m" (v->counter) : "memory");
	return c != 0;
}

/**
 * atomic_inc_and_test - increment and test 
 * @param v pointer of type spy_atomic_t
 * 
 * Atomically increments v by 1
 * and returns true if the result is zero, or false for all
 * other cases.
 */ 
static inline int spy_atomic_inc_and_test(spy_atomic_t *v)
{
	unsigned char c;

	__asm__ __volatile__(
		LOCK "incl %0; sete %1"
		:"=m" (v->counter), "=qm" (c)
		:"m" (v->counter) : "memory");
	return c != 0;
}

/**
 * atomic_add_negative - add and test if negative
 * @param v pointer of type spy_atomic_t
 * @param i integer value to add
 * 
 * Atomically adds i to v and returns true
 * if the result is negative, or false when
 * result is greater than or equal to zero.
 */ 
static inline int spy_atomic_add_negative(int i, spy_atomic_t *v)
{
	unsigned char c;

	__asm__ __volatile__(
		LOCK "addl %2,%0; sets %1"
		:"=m" (v->counter), "=qm" (c)
		:"ir" (i), "m" (v->counter) : "memory");
	return c;
}

static inline int spy_atomic_cmp_and_set(spy_atomic_t *lock, int old, int set)
{
    unsigned char res;

    __asm__ volatile (

		"lock;"
    "    cmpxchgl  %3, %1;   "
    "    sete      %0;       "

    : "=a" (res) : "m" (lock->counter), "a" (old), "r" (set) : "cc", "memory");

    return res;
}

#endif
