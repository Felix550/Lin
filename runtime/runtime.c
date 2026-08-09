#include <stdlib.h>
#include <string.h>
#define ARENA_IMPLEMENTATION
#include "arena.h" // https://github.com/tsoding

Arena arena = {0};

char* lin_strcat(char* a, char* b)
{
    size_t la = strlen(a);
    size_t lb = strlen(b);

    char* result = arena_alloc(&arena,la + lb + 1);

    arena_memcpy(result, a, la);
    arena_memcpy(result + la, b, lb);

    result[la + lb] = 0;

    return result;
}

char* wtoa(int a)
{
    return arena_sprintf(&arena,"%d",a);
}

char* ltoa(long a)
{
    return arena_sprintf(&arena,"%ld",a);
}

char* stoa(float a)
{
    return arena_sprintf(&arena,"%f",(double)a);
}

char* dtoa(double a)
{
    return arena_sprintf(&arena,"%f",a);
}

void lin_arena_free(void){
    arena_free(&arena);
}